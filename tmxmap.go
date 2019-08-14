package tmxmap

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	horizontalFlip = 0x80000000
	verticalFlip   = 0x40000000
	diagonalFlip   = 0x20000000
)

var NilTile = &TileInfo{Nil: true}

type GID uint32

// Map represents the TMX Map Format https://doc.mapeditor.org/en/stable/reference/tmx-map-format/
type Map struct {
	Version         string        `xml:"version,attr"`
	TiledVersion    string        `xml:"tiledversion,attr"`
	Orientation     string        `xml:"orientation,attr"`
	RenderOrder     string        `xml:"renderorder,attr"`
	Width           int           `xml:"width,attr"`
	Height          int           `xml:"height,attr"`
	TileWidth       int           `xml:"tilewidth,attr"`
	TileHeight      int           `xml:"tileheight,attr"`
	HexSideLength   int           `xml:"hexsidelength,attr"`
	StaggerAxis     int           `xml:"staggeraxis,attr"`
	StaggerIndex    int           `xml:"staggerindex,attr"`
	BackgroundColor string        `xml:"backgroundcolor,attr"`
	NextLayerID     int           `xml:"nextlayerid,attr"`
	NextObjectID    int           `xml:"nextobjectid,attr"`
	Properties      []Property    `xml:"properties>property"`
	TileSets        []TileSet     `xml:"tileset"`
	Layers          []Layer       `xml:"layer"`
	ObjectGroups    []ObjectGroup `xml:"objectgroup"`
}

type Property struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type TileSet struct {
	FirstGID   GID        `xml:"firstgid,attr"`
	Source     string     `xml:"source,attr"`
	Name       string     `xml:"name,attr"`
	TileWidth  int        `xml:"tilewidth,attr"`
	TileHeight int        `xml:"tileheight,attr"`
	Spacing    int        `xml:"spacing,attr"`
	Margin     int        `xml:"margin,attr"`
	Properties []Property `xml:"properties>property"`
	Image      Image      `xml:"image"`
	Tiles      []Tile     `xml:"tile"`
	Tilecount  int        `xml:"tilecount,attr"`
	Columns    int        `xml:"columns,attr"`
}

type Image struct {
	Source string `xml:"source,attr"`
	Trans  string `xml:"trans,attr"`
	Width  int    `xml:"width,attr"`
	Height int    `xml:"height,attr"`
	Image  image.Image
}

type Tile struct {
	ID    GID   `xml:"id,attr"`
	Image Image `xml:"image"`
}

type TileInfo struct {
	ID             GID
	TileSet        *TileSet
	HorizontalFlip bool
	VerticalFlip   bool
	DiagonalFlip   bool
	Nil            bool
}

type Layer struct {
	ID         int        `xml:"id,attr"`
	Name       string     `xml:"name,attr"`
	X          int        `xml:"x,attr"`
	Y          int        `xml:"y,attr"`
	Width      int        `xml:"width,attr"`
	Height     int        `xml:"height,attr"`
	Opacity    float32    `xml:"opacity,attr"`
	Visible    bool       `xml:"visible,attr"`
	OffsetX    int        `xml:"offsetx,attr"`
	OffsetY    int        `xml:"offsety,attr"`
	Properties []Property `xml:"properties>property"`
	Data       Data       `xml:"data"`
	Tiles      []*TileInfo
}

type Data struct {
	Encoding    string     `xml:"encoding,attr"`
	Compression string     `xml:"compression,attr"`
	RawData     []byte     `xml:",innerxml"`
	DataTiles   []DataTile `xml:"tile"`
	Chunk       []Chunk    `xml:"chunk"`
}

type DataTile struct {
	GID GID `xml:"gid,attr"`
}

type Chunk struct {
	X         int        `xml:"x,attr"`
	Y         int        `xml:"y,attr"`
	Width     int        `xml:"width,attr"`
	Height    int        `xml:"height,attr"`
	DataTiles []DataTile `xml:"tile"`
}

type ObjectGroup struct {
	Name       string     `xml:"name,attr"`
	Color      string     `xml:"color,attr"`
	Opacity    float32    `xml:"opacity,attr"`
	Visible    bool       `xml:"visible,attr"`
	Properties []Property `xml:"properties>property"`
	Objects    []Object   `xml:"object"`
}

type Object struct {
	Name       string     `xml:"name,attr"`
	Type       string     `xml:"type,attr"`
	X          int        `xml:"x,attr"`
	Y          int        `xml:"y,attr"`
	Width      int        `xml:"width,attr"`
	Height     int        `xml:"height,attr"`
	GID        int        `xml:"gid,attr"`
	Visible    bool       `xml:"visible,attr"`
	Properties []Property `xml:"properties>property"`
	Polygons   []Polygon  `xml:"polygon"`
	PolyLines  []PolyLine `xml:"polyline"`
}

type Polygon struct {
	Points string `xml:"points,attr"`
}

type PolyLine struct {
	Points string `xml:"points,attr"`
}

func (l *Layer) decodeXML() ([]GID, error) {
	gids := make([]GID, l.Width*l.Height)
	for i := 0; i < len(gids); i++ {
		gids[i] = l.Data.DataTiles[i].GID
	}
	return gids, nil
}

func (l *Layer) decodeBase64() ([]GID, error) {
	sanitized := bytes.TrimSpace(l.Data.RawData)
	decoder := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(sanitized))

	var reader io.Reader
	var err error
	switch l.Data.Compression {
	case "":
		reader = decoder
	case "gzip":
		reader, err = gzip.NewReader(decoder)
		if err != nil {
			return nil, err
		}
	case "zlib":
		reader, err = zlib.NewReader(decoder)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported compression: %s", l.Data.Compression)
	}

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	gids := make([]GID, l.Width*l.Height)
	for i := 0; i < len(data)/4; i++ {
		gids[i] = GID(data[i*4]) +
			GID(data[i*4+1])<<8 +
			GID(data[i*4+2])<<16 +
			GID(data[i*4+3])<<24
	}
	return gids, nil
}

func (l *Layer) decodeCSV() ([]GID, error) {
	sanitized := strings.Map(func(r rune) rune {
		if (r >= '0' && r <= '9') || r == ',' {
			return r
		}
		return -1
	}, string(l.Data.RawData))

	tokens := strings.Split(sanitized, ",")

	gids := make([]GID, l.Width*l.Height)
	for i, token := range tokens {
		gid, err := strconv.Atoi(token)
		if err != nil {
			return nil, err
		}
		gids[i] = GID(gid)
	}

	return gids, nil
}

func (l *Layer) decode() ([]GID, error) {
	switch l.Data.Encoding {
	case "":
		return l.decodeXML()
	case "base64":
		return l.decodeBase64()
	case "csv":
		return l.decodeCSV()
	}
	return nil, fmt.Errorf("unsupported encoding: %s", l.Data.Encoding)
}

func (i *Image) decode(baseDir string) error {
	file, err := os.Open(filepath.Join(baseDir, i.Source))
	if err != nil {
		return err
	}
	defer file.Close()

	i.Image, _, err = image.Decode(file)
	if err != nil {
		return err
	}
	return nil
}

func (ts *TileSet) decode(baseDir string) error {
	if ts.Source == "" {
		return nil
	}
	file, err := os.Open(filepath.Join(baseDir, ts.Source))
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := xml.NewDecoder(file)
	if err := decoder.Decode(ts); err != nil {
		return err
	}
	if err := ts.Image.decode(baseDir); err != nil {
		return err
	}
	return nil
}

func (m *Map) decodeGID(gid GID) (*TileInfo, error) {
	if gid == 0 {
		return NilTile, nil
	}

	clearGID := gid &^ (horizontalFlip | verticalFlip | diagonalFlip)
	for i := len(m.TileSets) - 1; i >= 0; i-- {
		if m.TileSets[i].FirstGID <= clearGID {
			return &TileInfo{
				ID:             clearGID - m.TileSets[i].FirstGID,
				TileSet:        &m.TileSets[i],
				HorizontalFlip: gid&horizontalFlip != 0,
				VerticalFlip:   gid&verticalFlip != 0,
				DiagonalFlip:   gid&diagonalFlip != 0,
				Nil:            gid == 0,
			}, nil
		}
	}

	return nil, fmt.Errorf("invalid tile GID: %d\n", gid)
}

func (m *Map) decode(baseDir string) error {
	for i := range m.TileSets {
		if err := m.TileSets[i].decode(baseDir); err != nil {
			return err
		}
	}
	for i := range m.Layers {
		layer := &m.Layers[i]
		gids, err := layer.decode()
		if err != nil {
			return err
		}

		layer.Tiles = make([]*TileInfo, len(gids))
		for j := 0; j < len(layer.Tiles); j++ {
			layer.Tiles[j], err = m.decodeGID(gids[j])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Load
func Load(name string) (*Map, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	baseDir, err := filepath.Abs(filepath.Dir(name))
	if err != nil {
		return nil, err
	}

	tmx := &Map{}
	decoder := xml.NewDecoder(file)
	if err := decoder.Decode(tmx); err != nil {
		return nil, err
	}
	if err := tmx.decode(baseDir); err != nil {
		return nil, err
	}
	return tmx, nil
}
