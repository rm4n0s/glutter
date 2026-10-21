//go:build linux

package vulkan

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const vulkanHeader = "/usr/include/vulkan/vulkan_core.h"

// runC compiles the C body (which has main's statements) against vulkan_core.h and
// returns what it prints. It skips the test when there is no compiler or no header.
func runC(t *testing.T, body string) string {
	t.Helper()
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("there is no C compiler (sudo dnf install gcc): the header is not checked")
	}
	if _, err := os.Stat(vulkanHeader); err != nil {
		t.Skip("vulkan_core.h is not installed (sudo dnf install vulkan-headers)")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "check.c")
	program := "#include <stdio.h>\n#include <stddef.h>\n#include <vulkan/vulkan_core.h>\nint main(void) {\n" + body + "return 0;\n}\n"
	if err := os.WriteFile(src, []byte(program), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "check")
	if out, err := exec.Command(cc, "-o", exe, src).CombinedOutput(); err != nil {
		t.Fatalf("the C program does not compile: %v\n%s\n%s", err, out, program)
	}
	out, err := exec.Command(exe).Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// cNames are the fields of the Go mirrors whose name is a Go keyword or is taken in
// the mirror; every other field has the C name.
var cNames = map[string]string{"typ": "type", "rng": "range"}

func TestStructLayouts(t *testing.T) {
	tests := []struct {
		c   string
		go_ any
	}{
		{"VkApplicationInfo", applicationInfo{}},
		{"VkInstanceCreateInfo", instanceCreateInfo{}},
		{"VkExtensionProperties", extensionProperties{}},
		{"VkLayerProperties", layerProperties{}},
		{"VkPhysicalDeviceLimits", physicalDeviceLimits{}},
		{"VkPhysicalDeviceProperties", physicalDeviceProperties{}},
		{"VkExtent3D", extent3D{}},
		{"VkExtent2D", extent2D{}},
		{"VkOffset2D", offset2D{}},
		{"VkOffset3D", offset3D{}},
		{"VkRect2D", rect2D{}},
		{"VkViewport", viewport{}},
		{"VkQueueFamilyProperties", queueFamilyProperties{}},
		{"VkMemoryType", memoryType{}},
		{"VkMemoryHeap", memoryHeap{}},
		{"VkPhysicalDeviceMemoryProperties", physicalDeviceMemoryProperties{}},
		{"VkPhysicalDeviceFeatures2", physicalDeviceFeatures2{}},
		{"VkPhysicalDeviceDynamicRenderingFeatures", physicalDeviceDynamicRenderingFeatures{}},
		{"VkPhysicalDeviceSynchronization2Features", physicalDeviceSynchronization2Features{}},
		{"VkDeviceQueueCreateInfo", deviceQueueCreateInfo{}},
		{"VkDeviceCreateInfo", deviceCreateInfo{}},
		{"VkBufferCreateInfo", bufferCreateInfo{}},
		{"VkMemoryRequirements", memoryRequirements{}},
		{"VkMemoryAllocateInfo", memoryAllocateInfo{}},
		{"VkImageCreateInfo", imageCreateInfo{}},
		{"VkComponentMapping", componentMapping{}},
		{"VkImageSubresourceRange", imageSubresourceRange{}},
		{"VkImageSubresourceLayers", imageSubresourceLayers{}},
		{"VkImageViewCreateInfo", imageViewCreateInfo{}},
		{"VkSamplerCreateInfo", samplerCreateInfo{}},
		{"VkShaderModuleCreateInfo", shaderModuleCreateInfo{}},
		{"VkDescriptorSetLayoutBinding", descriptorSetLayoutBinding{}},
		{"VkDescriptorSetLayoutCreateInfo", descriptorSetLayoutCreateInfo{}},
		{"VkPipelineLayoutCreateInfo", pipelineLayoutCreateInfo{}},
		{"VkPipelineShaderStageCreateInfo", pipelineShaderStageCreateInfo{}},
		{"VkVertexInputBindingDescription", vertexInputBindingDescription{}},
		{"VkVertexInputAttributeDescription", vertexInputAttributeDescription{}},
		{"VkPipelineVertexInputStateCreateInfo", pipelineVertexInputStateCreateInfo{}},
		{"VkPipelineInputAssemblyStateCreateInfo", pipelineInputAssemblyStateCreateInfo{}},
		{"VkPipelineViewportStateCreateInfo", pipelineViewportStateCreateInfo{}},
		{"VkPipelineRasterizationStateCreateInfo", pipelineRasterizationStateCreateInfo{}},
		{"VkPipelineMultisampleStateCreateInfo", pipelineMultisampleStateCreateInfo{}},
		{"VkPipelineColorBlendAttachmentState", pipelineColorBlendAttachmentState{}},
		{"VkPipelineColorBlendStateCreateInfo", pipelineColorBlendStateCreateInfo{}},
		{"VkPipelineDynamicStateCreateInfo", pipelineDynamicStateCreateInfo{}},
		{"VkPipelineRenderingCreateInfo", pipelineRenderingCreateInfo{}},
		{"VkGraphicsPipelineCreateInfo", graphicsPipelineCreateInfo{}},
		{"VkDescriptorPoolSize", descriptorPoolSize{}},
		{"VkDescriptorPoolCreateInfo", descriptorPoolCreateInfo{}},
		{"VkDescriptorSetAllocateInfo", descriptorSetAllocateInfo{}},
		{"VkDescriptorBufferInfo", descriptorBufferInfo{}},
		{"VkDescriptorImageInfo", descriptorImageInfo{}},
		{"VkWriteDescriptorSet", writeDescriptorSet{}},
		{"VkCommandPoolCreateInfo", commandPoolCreateInfo{}},
		{"VkCommandBufferAllocateInfo", commandBufferAllocateInfo{}},
		{"VkCommandBufferBeginInfo", commandBufferBeginInfo{}},
		{"VkFenceCreateInfo", fenceCreateInfo{}},
		{"VkSubmitInfo", submitInfo{}},
		{"VkClearValue", clearValue{}},
		{"VkRenderingAttachmentInfo", renderingAttachmentInfo{}},
		{"VkRenderingInfo", renderingInfo{}},
		{"VkMemoryBarrier2", memoryBarrier2{}},
		{"VkImageMemoryBarrier2", imageMemoryBarrier2{}},
		{"VkDependencyInfo", dependencyInfo{}},
		{"VkBufferImageCopy", bufferImageCopy{}},
		{"VkImageCopy", imageCopy{}},
		{"VkDebugUtilsMessengerCreateInfoEXT", debugUtilsMessengerCreateInfo{}},
		{"VkDebugUtilsMessengerCallbackDataEXT", debugUtilsMessengerCallbackData{}},
	}
	var body strings.Builder
	for _, tt := range tests {
		fmt.Fprintf(&body, "printf(\"%s size %%zu\\n\", sizeof(%s));\n", tt.c, tt.c)
		typ := reflect.TypeOf(tt.go_)
		for i := range typ.NumField() {
			name := typ.Field(i).Name
			if name == "_" {
				continue
			}
			c := name
			if r, ok := cNames[name]; ok {
				c = r
			}
			fmt.Fprintf(&body, "printf(\"%s.%s offset %%zu\\n\", offsetof(%s, %s));\n", tt.c, name, tt.c, c)
		}
	}
	got := map[string]uintptr{}
	for _, line := range strings.Split(runC(t, body.String()), "\n") {
		f := strings.Fields(line)
		if len(f) != 3 {
			continue
		}
		n, err := strconv.ParseUint(f[2], 10, 64)
		if err != nil {
			t.Fatalf("bad line %q", line)
		}
		got[f[0]+" "+f[1]] = uintptr(n)
	}
	for _, tt := range tests {
		typ := reflect.TypeOf(tt.go_)
		if want, ok := got[tt.c+" size"]; !ok || typ.Size() != want {
			t.Errorf("%s: the Go mirror has size %d, the C struct %d", tt.c, typ.Size(), want)
		}
		for i := range typ.NumField() {
			f := typ.Field(i)
			if f.Name == "_" {
				continue
			}
			want, ok := got[tt.c+"."+f.Name+" offset"]
			if !ok || f.Offset != want {
				t.Errorf("%s.%s: the Go offset is %d, the C offset %d", tt.c, f.Name, f.Offset, want)
			}
		}
	}
}

// TestConstantsMatchHeader compiles a program that prints every constant of
// consts.go that has a header name in its comment, and compares the values.
func TestConstantsMatchHeader(t *testing.T) {
	src, err := os.ReadFile("consts.go")
	if err != nil {
		t.Fatal(err)
	}
	line := regexp.MustCompile(`(?m)^\s*(\w+)\s*=\s*(-?(?:0x[0-9A-Fa-f]+|\d+))\s*// (VK_\w+)`)
	matches := line.FindAllStringSubmatch(string(src), -1)
	if len(matches) < 120 {
		t.Fatalf("parsed only %d constants from consts.go", len(matches))
	}
	var body bytes.Buffer
	for _, m := range matches {
		fmt.Fprintf(&body, "printf(\"%s %%lld\\n\", (long long)(%s));\n", m[3], m[3])
	}
	printed := map[string]int64{}
	for _, l := range strings.Split(runC(t, body.String()), "\n") {
		f := strings.Fields(l)
		if len(f) != 2 {
			continue
		}
		v, err := strconv.ParseInt(f[1], 10, 64)
		if err != nil {
			t.Fatalf("bad line %q", l)
		}
		printed[f[0]] = v
	}
	for _, m := range matches {
		var v int64
		if u, err := strconv.ParseUint(strings.TrimPrefix(m[2], "-"), 0, 64); err != nil {
			t.Fatalf("%s: %v", m[1], err)
		} else {
			v = int64(u)
			if strings.HasPrefix(m[2], "-") {
				v = -v
			}
		}
		got, ok := printed[m[3]]
		if !ok {
			t.Errorf("%s: the header has no %s", m[1], m[3])
		} else if got != v {
			t.Errorf("%s = %d, but %s is %d in the header", m[1], v, m[3], got)
		}
	}
}
