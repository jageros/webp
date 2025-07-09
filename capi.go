// Copyright 2014 <chaishushan{AT}gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//
// cgo pointer:
//
// Go1.3: Changes to the garbage collector
// http://golang.org/doc/go1.3#garbage_collector
//
// Go1.6:
// https://github.com/golang/proposal/blob/master/design/12416-cgo-pointers.md
//

package webp

/*
#cgo CFLAGS: -I./internal/libwebp-1.4.0/
#cgo CFLAGS: -I./internal/libwebp-1.4.0/src/
#cgo CFLAGS: -I./internal/include/
#cgo CFLAGS: -Wno-pointer-sign -DWEBP_USE_THREAD
#cgo !windows LDFLAGS: -lm

#include "webp.h"

#include <webp/decode.h>
#include <webp/encode.h>
#include <webp/mux.h>
#include <webp/demux.h>
#include <webp/mux_types.h>
#include <webp/types.h>

#include <stdlib.h>
#include <string.h>
#include <stdint.h>

// 检查是否为动画WebP
int webpIsAnimated(const uint8_t* data, size_t data_size) {
    WebPData webp_data = {data, data_size};
    WebPDemuxer* demux = WebPDemux(&webp_data);
    if (!demux) return 0;
    
    uint32_t frame_count = WebPDemuxGetI(demux, WEBP_FF_FRAME_COUNT);
    int is_animated = frame_count > 1;
    
    WebPDemuxDelete(demux);
    return is_animated;
}

// 获取动画信息
int webpGetAnimInfo(const uint8_t* data, size_t data_size, 
                   int* canvas_width, int* canvas_height, 
                   int* frame_count, int* loop_count) {
    WebPData webp_data = {data, data_size};
    WebPDemuxer* demux = WebPDemux(&webp_data);
    if (!demux) return 0;
    
    *canvas_width = WebPDemuxGetI(demux, WEBP_FF_CANVAS_WIDTH);
    *canvas_height = WebPDemuxGetI(demux, WEBP_FF_CANVAS_HEIGHT);
    *frame_count = WebPDemuxGetI(demux, WEBP_FF_FRAME_COUNT);
    *loop_count = WebPDemuxGetI(demux, WEBP_FF_LOOP_COUNT);
    
    WebPDemuxDelete(demux);
    return 1;
}

// 解码动画WebP的第一帧
uint8_t* webpDecodeAnimFirstFrame(const uint8_t* data, size_t data_size, 
                                 int* width, int* height) {
    WebPData webp_data = {data, data_size};
    WebPDemuxer* demux = WebPDemux(&webp_data);
    if (!demux) return NULL;
    
    WebPIterator iter;
    if (!WebPDemuxGetFrame(demux, 1, &iter)) {
        WebPDemuxDelete(demux);
        return NULL;
    }
    
    // 解码第一帧
    uint8_t* rgba = WebPDecodeRGBA(iter.fragment.bytes, iter.fragment.size, width, height);
    
    WebPDemuxReleaseIterator(&iter);
    WebPDemuxDelete(demux);
    
    return rgba;
}

// 解码动画WebP的所有帧
int webpDecodeAnimFrames(const uint8_t* data, size_t data_size,
                        uint8_t*** frames, int** timestamps, int** durations,
                        int* frame_count, int* width, int* height) {
    WebPData webp_data = {data, data_size};
    WebPDemuxer* demux = WebPDemux(&webp_data);
    if (!demux) return 0;
    
    *frame_count = WebPDemuxGetI(demux, WEBP_FF_FRAME_COUNT);
    *width = WebPDemuxGetI(demux, WEBP_FF_CANVAS_WIDTH);
    *height = WebPDemuxGetI(demux, WEBP_FF_CANVAS_HEIGHT);
    
    if (*frame_count <= 0) {
        WebPDemuxDelete(demux);
        return 0;
    }
    
    // 分配内存
    *frames = (uint8_t**)malloc(*frame_count * sizeof(uint8_t*));
    *timestamps = (int*)malloc(*frame_count * sizeof(int));
    *durations = (int*)malloc(*frame_count * sizeof(int));
    
    if (!*frames || !*timestamps || !*durations) {
        if (*frames) free(*frames);
        if (*timestamps) free(*timestamps);
        if (*durations) free(*durations);
        WebPDemuxDelete(demux);
        return 0;
    }
    
    WebPIterator iter;
    int frame_idx = 0;
    
    if (WebPDemuxGetFrame(demux, 1, &iter)) {
        do {
            if (frame_idx >= *frame_count) break;
            
            int frame_width, frame_height;
            uint8_t* rgba = WebPDecodeRGBA(iter.fragment.bytes, iter.fragment.size, 
                                          &frame_width, &frame_height);
            
            if (rgba) {
                (*frames)[frame_idx] = rgba;
                (*timestamps)[frame_idx] = frame_idx * 100; // 简单的时间戳计算
                (*durations)[frame_idx] = iter.duration;
                frame_idx++;
            }
        } while (WebPDemuxNextFrame(&iter));
        
        WebPDemuxReleaseIterator(&iter);
    }
    
    WebPDemuxDelete(demux);
    return frame_idx; // 返回实际解码的帧数
}

// 释放帧数据
void webpFreeFrames(uint8_t** frames, int* timestamps, int* durations, int frame_count) {
    if (frames) {
        for (int i = 0; i < frame_count; i++) {
            if (frames[i]) free(frames[i]);
        }
        free(frames);
    }
    if (timestamps) free(timestamps);
    if (durations) free(durations);
}
*/
import "C"
import (
	"errors"
	"image"
	"unsafe"
)

func webpGetInfo(data []byte) (width, height int, hasAlpha bool, err error) {
	if len(data) == 0 {
		err = errors.New("webpGetInfo: bad arguments, data is empty")
		return
	}
	if len(data) > maxWebpHeaderSize {
		data = data[:maxWebpHeaderSize]
	}

	var features C.WebPBitstreamFeatures
	if C.WebPGetFeatures((*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)), &features) != C.VP8_STATUS_OK {
		err = errors.New("C.WebPGetFeatures: failed")
		return
	}
	width, height = int(features.width), int(features.height)
	hasAlpha = (features.has_alpha != 0)
	return
}

func webpDecodeGray(data []byte) (pix []byte, width, height int, err error) {
	if len(data) == 0 {
		err = errors.New("webpDecodeGray: bad arguments")
		return
	}

	var cw, ch C.int
	var cptr = C.webpDecodeGray((*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)), &cw, &ch)
	if cptr == nil {
		err = errors.New("webpDecodeGray: failed")
		return
	}
	defer C.free(unsafe.Pointer(cptr))

	pix = make([]byte, int(cw*ch*1))
	copy(pix, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(pix):len(pix)])
	width, height = int(cw), int(ch)
	return
}

func webpDecodeRGB(data []byte) (pix []byte, width, height int, err error) {
	if len(data) == 0 {
		err = errors.New("webpDecodeRGB: bad arguments")
		return
	}

	var cw, ch C.int
	var cptr = C.webpDecodeRGB((*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)), &cw, &ch)
	if cptr == nil {
		err = errors.New("webpDecodeRGB: failed")
		return
	}
	defer C.free(unsafe.Pointer(cptr))

	pix = make([]byte, int(cw*ch*3))
	copy(pix, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(pix):len(pix)])
	width, height = int(cw), int(ch)
	return
}

func webpDecodeRGBA(data []byte) (pix []byte, width, height int, err error) {
	if len(data) == 0 {
		err = errors.New("webpDecodeRGBA: bad arguments")
		return
	}

	var cw, ch C.int
	var cptr = C.webpDecodeRGBA((*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)), &cw, &ch)
	if cptr == nil {
		err = errors.New("webpDecodeRGBA: failed")
		return
	}
	defer C.free(unsafe.Pointer(cptr))

	pix = make([]byte, int(cw*ch*4))
	copy(pix, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(pix):len(pix)])
	width, height = int(cw), int(ch)
	return
}

func webpDecodeGrayToSize(data []byte, width, height int) (pix []byte, err error) {
	pix = make([]byte, int(width*height))
	stride := C.int(width)
	res := C.webpDecodeGrayToSize((*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)), C.int(width), C.int(height), stride, (*C.uint8_t)(unsafe.Pointer(&pix[0])))
	if res != C.VP8_STATUS_OK {
		pix = nil
		err = errors.New("webpDecodeGrayToSize: failed")
	}
	return
}

func webpDecodeRGBToSize(data []byte, width, height int) (pix []byte, err error) {
	pix = make([]byte, int(3*width*height))
	stride := C.int(3 * width)
	res := C.webpDecodeRGBToSize((*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)), C.int(width), C.int(height), stride, (*C.uint8_t)(unsafe.Pointer(&pix[0])))
	if res != C.VP8_STATUS_OK {
		pix = nil
		err = errors.New("webpDecodeRGBToSize: failed")
	}
	return
}

func webpDecodeRGBAToSize(data []byte, width, height int) (pix []byte, err error) {
	pix = make([]byte, int(4*width*height))
	stride := C.int(4 * width)
	res := C.webpDecodeRGBAToSize((*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)), C.int(width), C.int(height), stride, (*C.uint8_t)(unsafe.Pointer(&pix[0])))
	if res != C.VP8_STATUS_OK {
		pix = nil
		err = errors.New("webpDecodeRGBAToSize: failed")
	}
	return
}

func webpEncodeGray(pix []byte, width, height, stride int, quality float32) (output []byte, err error) {
	if len(pix) == 0 || width <= 0 || height <= 0 || stride <= 0 || quality < 0.0 {
		err = errors.New("webpEncodeGray: bad arguments")
		return
	}
	if stride < width*1 && len(pix) < height*stride {
		err = errors.New("webpEncodeGray: bad arguments")
		return
	}

	var cptr_size C.size_t
	var cptr = C.webpEncodeGray(
		(*C.uint8_t)(unsafe.Pointer(&pix[0])), C.int(width), C.int(height),
		C.int(stride), C.float(quality),
		&cptr_size,
	)
	if cptr == nil || cptr_size == 0 {
		err = errors.New("webpEncodeGray: failed")
		return
	}
	defer C.free(unsafe.Pointer(cptr))

	output = make([]byte, int(cptr_size))
	copy(output, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(output):len(output)])
	return
}

func webpEncodeRGB(pix []byte, width, height, stride int, quality float32) (output []byte, err error) {
	if len(pix) == 0 || width <= 0 || height <= 0 || stride <= 0 || quality < 0.0 {
		err = errors.New("webpEncodeRGB: bad arguments")
		return
	}
	if stride < width*3 && len(pix) < height*stride {
		err = errors.New("webpEncodeRGB: bad arguments")
		return
	}

	var cptr_size C.size_t
	var cptr = C.webpEncodeRGB(
		(*C.uint8_t)(unsafe.Pointer(&pix[0])), C.int(width), C.int(height),
		C.int(stride), C.float(quality),
		&cptr_size,
	)
	if cptr == nil || cptr_size == 0 {
		err = errors.New("webpEncodeRGB: failed")
		return
	}
	defer C.free(unsafe.Pointer(cptr))

	output = make([]byte, int(cptr_size))
	copy(output, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(output):len(output)])
	return
}

func webpEncodeRGBA(pix []byte, width, height, stride int, quality float32) (output []byte, err error) {
	if len(pix) == 0 || width <= 0 || height <= 0 || stride <= 0 || quality < 0.0 {
		err = errors.New("webpEncodeRGBA: bad arguments")
		return
	}
	if stride < width*4 && len(pix) < height*stride {
		err = errors.New("webpEncodeRGBA: bad arguments")
		return
	}

	var cptr_size C.size_t
	var cptr = C.webpEncodeRGBA(
		(*C.uint8_t)(unsafe.Pointer(&pix[0])), C.int(width), C.int(height),
		C.int(stride), C.float(quality),
		&cptr_size,
	)
	if cptr == nil || cptr_size == 0 {
		err = errors.New("webpEncodeRGBA: failed")
		return
	}
	defer C.free(unsafe.Pointer(cptr))

	output = make([]byte, int(cptr_size))
	copy(output, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(output):len(output)])
	return
}

func webpEncodeLosslessGray(pix []byte, width, height, stride int) (output []byte, err error) {
	if len(pix) == 0 || width <= 0 || height <= 0 || stride <= 0 {
		err = errors.New("webpEncodeLosslessGray: bad arguments")
		return
	}
	if stride < width*1 && len(pix) < height*stride {
		err = errors.New("webpEncodeLosslessGray: bad arguments")
		return
	}

	var cptr_size C.size_t
	var cptr = C.webpEncodeLosslessGray(
		(*C.uint8_t)(unsafe.Pointer(&pix[0])), C.int(width), C.int(height),
		C.int(stride),
		&cptr_size,
	)
	if cptr == nil || cptr_size == 0 {
		err = errors.New("webpEncodeLosslessGray: failed")
		return
	}
	defer C.free(unsafe.Pointer(cptr))

	output = make([]byte, int(cptr_size))
	copy(output, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(output):len(output)])
	return
}

func webpEncodeLosslessRGB(pix []byte, width, height, stride int) (output []byte, err error) {
	if len(pix) == 0 || width <= 0 || height <= 0 || stride <= 0 {
		err = errors.New("webpEncodeLosslessRGB: bad arguments")
		return
	}
	if stride < width*3 && len(pix) < height*stride {
		err = errors.New("webpEncodeLosslessRGB: bad arguments")
		return
	}

	var cptr_size C.size_t
	var cptr = C.webpEncodeLosslessRGB(
		(*C.uint8_t)(unsafe.Pointer(&pix[0])), C.int(width), C.int(height),
		C.int(stride),
		&cptr_size,
	)
	if cptr == nil || cptr_size == 0 {
		err = errors.New("webpEncodeLosslessRGB: failed")
		return
	}
	defer C.free(unsafe.Pointer(cptr))

	output = make([]byte, int(cptr_size))
	copy(output, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(output):len(output)])
	return
}

func webpEncodeLosslessRGBA(exact int, pix []byte, width, height, stride int) (output []byte, err error) {
	if len(pix) == 0 || width <= 0 || height <= 0 || stride <= 0 {
		err = errors.New("webpEncodeLosslessRGBA: bad arguments")
		return
	}
	if stride < width*4 && len(pix) < height*stride {
		err = errors.New("webpEncodeLosslessRGBA: bad arguments")
		return
	}

	var cptr_size C.size_t
	var cptr = C.webpEncodeLosslessRGBA(
		C.int(exact), (*C.uint8_t)(unsafe.Pointer(&pix[0])), C.int(width), C.int(height),
		C.int(stride),
		&cptr_size,
	)
	if cptr == nil || cptr_size == 0 {
		err = errors.New("webpEncodeLosslessRGBA: failed")
		return
	}
	defer C.free(unsafe.Pointer(cptr))

	output = make([]byte, int(cptr_size))
	copy(output, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(output):len(output)])
	return
}

func webpGetEXIF(data []byte) (metadata []byte, err error) {
	if len(data) == 0 {
		err = errors.New("webpGetEXIF: bad arguments")
		return
	}

	var cptr_size C.size_t
	var cptr = C.webpGetEXIF(
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		&cptr_size,
	)
	if cptr == nil || cptr_size == 0 {
		err = errors.New("webpGetEXIF: failed")
		return
	}
	defer C.free(unsafe.Pointer(cptr))

	metadata = make([]byte, int(cptr_size))
	copy(metadata, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(metadata):len(metadata)])
	return
}
func webpGetICCP(data []byte) (metadata []byte, err error) {
	if len(data) == 0 {
		err = errors.New("webpGetICCP: bad arguments")
		return
	}

	var cptr_size C.size_t
	var cptr = C.webpGetICCP(
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		&cptr_size,
	)
	if cptr == nil || cptr_size == 0 {
		err = errors.New("webpGetICCP: failed")
		return
	}
	defer C.free(unsafe.Pointer(cptr))

	metadata = make([]byte, int(cptr_size))
	copy(metadata, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(metadata):len(metadata)])
	return
}
func webpGetXMP(data []byte) (metadata []byte, err error) {
	if len(data) == 0 {
		err = errors.New("webpGetXMP: bad arguments")
		return
	}

	var cptr_size C.size_t
	var cptr = C.webpGetXMP(
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		&cptr_size,
	)
	if cptr == nil || cptr_size == 0 {
		err = errors.New("webpGetXMP: failed")
		return
	}
	defer C.free(unsafe.Pointer(cptr))

	metadata = make([]byte, int(cptr_size))
	copy(metadata, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(metadata):len(metadata)])
	return
}
func webpGetMetadata(data []byte, format string) (metadata []byte, err error) {
	if len(data) == 0 {
		err = errors.New("webpGetMetadata: bad arguments")
		return
	}

	switch format {
	case "EXIF":
		return webpGetEXIF(data)
	case "ICCP":
		return webpGetICCP(data)
	case "XMP":
		return webpGetXMP(data)
	default:
		err = errors.New("webpGetMetadata: unknown format")
		return
	}
}

func webpSetEXIF(data, metadata []byte) (newData []byte, err error) {
	if len(data) == 0 || len(metadata) == 0 {
		err = errors.New("webpSetEXIF: bad arguments")
		return
	}

	var cptr_size C.size_t
	var cptr = C.webpSetEXIF(
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		(*C.char)(unsafe.Pointer(&metadata[0])), C.size_t(len(metadata)),
		&cptr_size,
	)
	if cptr == nil || cptr_size == 0 {
		err = errors.New("webpSetEXIF: failed")
		return
	}
	defer C.free(unsafe.Pointer(cptr))

	newData = make([]byte, int(cptr_size))
	copy(newData, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(newData):len(newData)])
	return
}
func webpSetICCP(data, metadata []byte) (newData []byte, err error) {
	if len(data) == 0 || len(metadata) == 0 {
		err = errors.New("webpSetICCP: bad arguments")
		return
	}

	var cptr_size C.size_t
	var cptr = C.webpSetICCP(
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		(*C.char)(unsafe.Pointer(&metadata[0])), C.size_t(len(metadata)),
		&cptr_size,
	)
	if cptr == nil || cptr_size == 0 {
		err = errors.New("webpSetICCP: failed")
		return
	}
	defer C.free(unsafe.Pointer(cptr))

	newData = make([]byte, int(cptr_size))
	copy(newData, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(newData):len(newData)])
	return
}
func webpSetXMP(data, metadata []byte) (newData []byte, err error) {
	if len(data) == 0 || len(metadata) == 0 {
		err = errors.New("webpSetXMP: bad arguments")
		return
	}

	var cptr_size C.size_t
	var cptr = C.webpSetXMP(
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		(*C.char)(unsafe.Pointer(&metadata[0])), C.size_t(len(metadata)),
		&cptr_size,
	)
	if cptr == nil || cptr_size == 0 {
		err = errors.New("webpSetXMP: failed")
		return
	}
	defer C.free(unsafe.Pointer(cptr))

	newData = make([]byte, int(cptr_size))
	copy(newData, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(newData):len(newData)])
	return
}
func webpSetMetadata(data, metadata []byte, format string) (newData []byte, err error) {
	if len(data) == 0 || len(metadata) == 0 {
		err = errors.New("webpSetMetadata: bad arguments")
		return
	}

	switch format {
	case "EXIF":
		return webpSetEXIF(data, metadata)
	case "ICCP":
		return webpSetICCP(data, metadata)
	case "XMP":
		return webpSetXMP(data, metadata)
	default:
		err = errors.New("webpSetMetadata: unknown format")
		return
	}
}

func webpDelEXIF(data []byte) (newData []byte, err error) {
	if len(data) == 0 {
		err = errors.New("webpDelEXIF: bad arguments")
		return
	}

	var cptr_size C.size_t
	var cptr = C.webpDelEXIF(
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		&cptr_size,
	)
	if cptr == nil || cptr_size == 0 {
		err = errors.New("webpDelEXIF: failed")
		return
	}
	defer C.free(unsafe.Pointer(cptr))

	newData = make([]byte, int(cptr_size))
	copy(newData, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(newData):len(newData)])
	return
}
func webpDelICCP(data []byte) (newData []byte, err error) {
	if len(data) == 0 {
		err = errors.New("webpDelICCP: bad arguments")
		return
	}

	var cptr_size C.size_t
	var cptr = C.webpDelICCP(
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		&cptr_size,
	)
	if cptr == nil || cptr_size == 0 {
		err = errors.New("webpDelICCP: failed")
		return
	}
	defer C.free(unsafe.Pointer(cptr))

	newData = make([]byte, int(cptr_size))
	copy(newData, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(newData):len(newData)])
	return
}
func webpDelXMP(data []byte) (newData []byte, err error) {
	if len(data) == 0 {
		err = errors.New("webpDelXMP: bad arguments")
		return
	}

	var cptr_size C.size_t
	var cptr = C.webpDelXMP(
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		&cptr_size,
	)
	if cptr == nil || cptr_size == 0 {
		err = errors.New("webpDelXMP: failed")
		return
	}
	defer C.free(unsafe.Pointer(cptr))

	newData = make([]byte, int(cptr_size))
	copy(newData, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(newData):len(newData)])
	return
}

// 动画WebP相关函数

func webpIsAnimated(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	
	result := C.webpIsAnimated((*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)))
	return result != 0
}

func webpGetAnimInfo(data []byte) (*AnimInfo, error) {
	if len(data) == 0 {
		return nil, errors.New("webpGetAnimInfo: data is empty")
	}
	
	var cCanvasWidth, cCanvasHeight, cFrameCount, cLoopCount C.int
	result := C.webpGetAnimInfo(
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		&cCanvasWidth, &cCanvasHeight, &cFrameCount, &cLoopCount,
	)
	
	if result == 0 {
		return nil, errors.New("webpGetAnimInfo: failed to get animation info")
	}
	
	return &AnimInfo{
		CanvasWidth:  int(cCanvasWidth),
		CanvasHeight: int(cCanvasHeight),
		FrameCount:   int(cFrameCount),
		LoopCount:    int(cLoopCount),
	}, nil
}

func webpDecodeAnimFirstFrame(data []byte) (*image.RGBA, error) {
	if len(data) == 0 {
		return nil, errors.New("webpDecodeAnimFirstFrame: data is empty")
	}
	
	var cWidth, cHeight C.int
	cptr := C.webpDecodeAnimFirstFrame(
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		&cWidth, &cHeight,
	)
	
	if cptr == nil {
		return nil, errors.New("webpDecodeAnimFirstFrame: failed to decode first frame")
	}
	defer C.free(unsafe.Pointer(cptr))
	
	width := int(cWidth)
	height := int(cHeight)
	pix := make([]byte, width*height*4)
	copy(pix, ((*[1 << 30]byte)(unsafe.Pointer(cptr)))[0:len(pix):len(pix)])
	
	return &image.RGBA{
		Pix:    pix,
		Stride: 4 * width,
		Rect:   image.Rect(0, 0, width, height),
	}, nil
}

func webpDecodeAnimFrames(data []byte) ([]*Frame, error) {
	if len(data) == 0 {
		return nil, errors.New("webpDecodeAnimFrames: data is empty")
	}
	
	var cFrames **C.uint8_t
	var cTimestamps, cDurations *C.int
	var cFrameCount, cWidth, cHeight C.int
	
	result := C.webpDecodeAnimFrames(
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		&cFrames, &cTimestamps, &cDurations,
		&cFrameCount, &cWidth, &cHeight,
	)
	
	if result == 0 {
		return nil, errors.New("webpDecodeAnimFrames: failed to decode frames")
	}
	
	defer C.webpFreeFrames(cFrames, cTimestamps, cDurations, cFrameCount)
	
	frameCount := int(cFrameCount)
	width := int(cWidth)
	height := int(cHeight)
	pixelCount := width * height * 4
	
	// 转换C数组到Go切片
	frames := make([]*Frame, frameCount)
	
	cFramesSlice := (*[1 << 20]*C.uint8_t)(unsafe.Pointer(cFrames))[:frameCount:frameCount]
	cTimestampsSlice := (*[1 << 20]C.int)(unsafe.Pointer(cTimestamps))[:frameCount:frameCount]
	cDurationsSlice := (*[1 << 20]C.int)(unsafe.Pointer(cDurations))[:frameCount:frameCount]
	
	for i := 0; i < frameCount; i++ {
		if cFramesSlice[i] != nil {
			pix := make([]byte, pixelCount)
			copy(pix, ((*[1 << 30]byte)(unsafe.Pointer(cFramesSlice[i])))[0:pixelCount:pixelCount])
			
			frames[i] = &Frame{
				Image: &image.RGBA{
					Pix:    pix,
					Stride: 4 * width,
					Rect:   image.Rect(0, 0, width, height),
				},
				Timestamp: int(cTimestampsSlice[i]),
				Duration:  int(cDurationsSlice[i]),
			}
		}
	}
	
	return frames, nil
}
