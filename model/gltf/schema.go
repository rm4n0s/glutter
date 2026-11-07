package gltf

import (
	"encoding/json"
	"strconv"
	"strings"
)

// The types of this file are the JSON of a glTF 2.0 file (https://registry.khronos.org/glTF/specs/2.0/glTF-2.0.html#reference-gltf);
// a field that has a default in the specification and that the loader needs to tell from a value is a pointer. A part of the file
// that the loader does not read (extras, the extensions that are not read) is not here.

type document struct {
	Asset              assetJSON                  `json:"asset"`
	ExtensionsUsed     []string                   `json:"extensionsUsed"`
	ExtensionsRequired []string                   `json:"extensionsRequired"`
	Extensions         map[string]json.RawMessage `json:"extensions"`
	Scene              *int                       `json:"scene"`
	Scenes             []sceneJSON                `json:"scenes"`
	Nodes              []nodeJSON                 `json:"nodes"`
	Meshes             []meshJSON                 `json:"meshes"`
	Materials          []materialJSON             `json:"materials"`
	Textures           []textureJSON              `json:"textures"`
	Images             []imageJSON                `json:"images"`
	Samplers           []samplerJSON              `json:"samplers"`
	Accessors          []accessorJSON             `json:"accessors"`
	BufferViews        []bufferViewJSON           `json:"bufferViews"`
	Buffers            []bufferJSON               `json:"buffers"`
	Cameras            []cameraJSON               `json:"cameras"`
	Skins              []skinJSON                 `json:"skins"`
	Animations         []animationJSON            `json:"animations"`
}

type assetJSON struct {
	Version    string `json:"version"`
	MinVersion string `json:"minVersion"`
	Generator  string `json:"generator"`
}

type sceneJSON struct {
	Name  string `json:"name"`
	Nodes []int  `json:"nodes"`
}

type nodeJSON struct {
	Name        string                     `json:"name"`
	Children    []int                      `json:"children"`
	Mesh        *int                       `json:"mesh"`
	Skin        *int                       `json:"skin"`
	Camera      *int                       `json:"camera"`
	Matrix      []float64                  `json:"matrix"`
	Translation []float64                  `json:"translation"`
	Rotation    []float64                  `json:"rotation"`
	Scale       []float64                  `json:"scale"`
	Weights     []float32                  `json:"weights"`
	Extensions  map[string]json.RawMessage `json:"extensions"`
}

type meshJSON struct {
	Name       string          `json:"name"`
	Primitives []primitiveJSON `json:"primitives"`
	Weights    []float32       `json:"weights"`
	Extras     json.RawMessage `json:"extras"`
}

type primitiveJSON struct {
	Attributes map[string]int             `json:"attributes"`
	Indices    *int                       `json:"indices"`
	Material   *int                       `json:"material"`
	Mode       *int                       `json:"mode"`
	Targets    []map[string]int           `json:"targets"`
	Extensions map[string]json.RawMessage `json:"extensions"`
}

type materialJSON struct {
	Name             string                     `json:"name"`
	PBR              *pbrJSON                   `json:"pbrMetallicRoughness"`
	NormalTexture    *textureInfoJSON           `json:"normalTexture"`
	OcclusionTexture *textureInfoJSON           `json:"occlusionTexture"`
	EmissiveTexture  *textureInfoJSON           `json:"emissiveTexture"`
	EmissiveFactor   []float32                  `json:"emissiveFactor"`
	AlphaMode        string                     `json:"alphaMode"`
	AlphaCutoff      *float32                   `json:"alphaCutoff"`
	DoubleSided      bool                       `json:"doubleSided"`
	Extensions       map[string]json.RawMessage `json:"extensions"`
}

type pbrJSON struct {
	BaseColorFactor          []float32        `json:"baseColorFactor"`
	BaseColorTexture         *textureInfoJSON `json:"baseColorTexture"`
	MetallicFactor           *float32         `json:"metallicFactor"`
	RoughnessFactor          *float32         `json:"roughnessFactor"`
	MetallicRoughnessTexture *textureInfoJSON `json:"metallicRoughnessTexture"`
}

// textureInfoJSON is textureInfo, normalTextureInfo and occlusionTextureInfo.
type textureInfoJSON struct {
	Index      *int                       `json:"index"`
	TexCoord   *int                       `json:"texCoord"`
	Scale      *float32                   `json:"scale"`
	Strength   *float32                   `json:"strength"`
	Extensions map[string]json.RawMessage `json:"extensions"`
}

type textureTransformJSON struct {
	Offset   []float32 `json:"offset"`
	Rotation *float32  `json:"rotation"`
	Scale    []float32 `json:"scale"`
	TexCoord *int      `json:"texCoord"`
}

type textureJSON struct {
	Name       string                     `json:"name"`
	Sampler    *int                       `json:"sampler"`
	Source     *int                       `json:"source"`
	Extensions map[string]json.RawMessage `json:"extensions"`
}

type imageJSON struct {
	Name       string `json:"name"`
	URI        string `json:"uri"`
	MimeType   string `json:"mimeType"`
	BufferView *int   `json:"bufferView"`
}

type samplerJSON struct {
	MagFilter *int `json:"magFilter"`
	MinFilter *int `json:"minFilter"`
	WrapS     *int `json:"wrapS"`
	WrapT     *int `json:"wrapT"`
}

type accessorJSON struct {
	BufferView    *int        `json:"bufferView"`
	ByteOffset    int         `json:"byteOffset"`
	ComponentType int         `json:"componentType"`
	Normalized    bool        `json:"normalized"`
	Count         int         `json:"count"`
	Type          string      `json:"type"`
	Sparse        *sparseJSON `json:"sparse"`
}

type sparseJSON struct {
	Count   int `json:"count"`
	Indices struct {
		BufferView    *int `json:"bufferView"`
		ByteOffset    int  `json:"byteOffset"`
		ComponentType int  `json:"componentType"`
	} `json:"indices"`
	Values struct {
		BufferView *int `json:"bufferView"`
		ByteOffset int  `json:"byteOffset"`
	} `json:"values"`
}

type bufferViewJSON struct {
	Buffer     *int                       `json:"buffer"`
	ByteOffset int                        `json:"byteOffset"`
	ByteLength *int                       `json:"byteLength"`
	ByteStride int                        `json:"byteStride"`
	Extensions map[string]json.RawMessage `json:"extensions"`
}

// meshoptJSON is the extension EXT_meshopt_compression of a buffer view.
type meshoptJSON struct {
	Buffer     *int   `json:"buffer"`
	ByteOffset int    `json:"byteOffset"`
	ByteLength *int   `json:"byteLength"`
	ByteStride *int   `json:"byteStride"`
	Mode       string `json:"mode"`
	Filter     string `json:"filter"`
	Count      *int   `json:"count"`
}

type bufferJSON struct {
	Name       string                     `json:"name"`
	URI        string                     `json:"uri"`
	ByteLength *int                       `json:"byteLength"`
	Extensions map[string]json.RawMessage `json:"extensions"`
}

type cameraJSON struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Perspective *struct {
		AspectRatio float64  `json:"aspectRatio"`
		YFov        *float64 `json:"yfov"`
		ZFar        float64  `json:"zfar"`
		ZNear       *float64 `json:"znear"`
	} `json:"perspective"`
	Orthographic *struct {
		XMag  *float64 `json:"xmag"`
		YMag  *float64 `json:"ymag"`
		ZFar  *float64 `json:"zfar"`
		ZNear *float64 `json:"znear"`
	} `json:"orthographic"`
}

type skinJSON struct {
	Name                string `json:"name"`
	InverseBindMatrices *int   `json:"inverseBindMatrices"`
	Skeleton            *int   `json:"skeleton"`
	Joints              []int  `json:"joints"`
}

type animationJSON struct {
	Name     string                 `json:"name"`
	Channels []animationChannelJSON `json:"channels"`
	Samplers []animationSamplerJSON `json:"samplers"`
}

type animationChannelJSON struct {
	Sampler *int `json:"sampler"`
	Target  struct {
		Node       *int                       `json:"node"`
		Path       string                     `json:"path"`
		Extensions map[string]json.RawMessage `json:"extensions"`
	} `json:"target"`
}

type animationSamplerJSON struct {
	Input         *int   `json:"input"`
	Output        *int   `json:"output"`
	Interpolation string `json:"interpolation"`
}

// lightsJSON is the extension KHR_lights_punctual of the file.
type lightsJSON struct {
	Lights []lightJSON `json:"lights"`
}

type lightJSON struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Color     []float32 `json:"color"`
	Intensity *float32  `json:"intensity"`
	Range     *float32  `json:"range"`
	Spot      *struct {
		InnerConeAngle *float32 `json:"innerConeAngle"`
		OuterConeAngle *float32 `json:"outerConeAngle"`
	} `json:"spot"`
}

// nodeLightJSON is the extension KHR_lights_punctual of a node.
type nodeLightJSON struct {
	Light *int `json:"light"`
}

// meshExtrasJSON is the extras of a mesh that name the morph targets (a convention of the exporters).
type meshExtrasJSON struct {
	TargetNames []string `json:"targetNames"`
}

// ptr makes a JSON pointer (RFC 6901) from the names and the indices of a path, such as "/meshes/0/primitives/1".
func ptr(parts ...any) string {
	var b strings.Builder
	for _, p := range parts {
		b.WriteByte('/')
		switch v := p.(type) {
		case int:
			b.WriteString(strconv.Itoa(v))
		case string:
			b.WriteString(strings.NewReplacer("~", "~0", "/", "~1").Replace(v))
		}
	}
	return b.String()
}
