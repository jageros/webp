// Copyright 2014 <chaishushan{AT}gmail.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build cgo
// +build cgo

package webp

/*
#cgo CFLAGS: -I./internal/libwebp-1.4.0/
#cgo CFLAGS: -I./internal/libwebp-1.4.0/src/
#cgo CFLAGS: -I./internal/include/
#cgo CFLAGS: -Wno-pointer-sign -DWEBP_USE_THREAD
#cgo !windows LDFLAGS: -lm

#include <webp/decode.h>
#include <webp/demux.h>
#include <webp/mux_types.h>
#include <stdlib.h>
#include <string.h>

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

// AnimInfo 包含动画WebP的基本信息
type AnimInfo struct {
	CanvasWidth  int // 画布宽度
	CanvasHeight int // 画布高度
	FrameCount   int // 帧数
	LoopCount    int // 循环次数
}

// Frame 表示动画中的一帧
type Frame struct {
	Image     *image.RGBA // 帧图像
	Timestamp int         // 时间戳（毫秒）
	Duration  int         // 持续时间（毫秒）
}

// IsAnimated 检查WebP数据是否为动画
func IsAnimated(data []byte) (bool, error) {
	if len(data) == 0 {
		return false, errors.New("IsAnimated: data is empty")
	}
	
	result := C.webpIsAnimated((*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)))
	return result != 0, nil
}

// GetAnimInfo 获取动画WebP的基本信息
func GetAnimInfo(data []byte) (*AnimInfo, error) {
	if len(data) == 0 {
		return nil, errors.New("GetAnimInfo: data is empty")
	}
	
	var canvasWidth, canvasHeight, frameCount, loopCount C.int
	result := C.webpGetAnimInfo(
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		&canvasWidth, &canvasHeight, &frameCount, &loopCount,
	)
	
	if result == 0 {
		return nil, errors.New("GetAnimInfo: failed to get animation info")
	}
	
	return &AnimInfo{
		CanvasWidth:  int(canvasWidth),
		CanvasHeight: int(canvasHeight),
		FrameCount:   int(frameCount),
		LoopCount:    int(loopCount),
	}, nil
}

// DecodeAnimFirstFrame 解码动画WebP的第一帧
func DecodeAnimFirstFrame(data []byte) (*image.RGBA, error) {
	if len(data) == 0 {
		return nil, errors.New("DecodeAnimFirstFrame: data is empty")
	}
	
	var width, height C.int
	cptr := C.webpDecodeAnimFirstFrame(
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		&width, &height,
	)
	
	if cptr == nil {
		return nil, errors.New("DecodeAnimFirstFrame: failed to decode first frame")
	}
	defer C.free(unsafe.Pointer(cptr))
	
	w, h := int(width), int(height)
	pixelCount := w * h * 4
	pixData := make([]byte, pixelCount)
	copy(pixData, (*[1 << 30]byte)(unsafe.Pointer(cptr))[:pixelCount:pixelCount])
	
	return &image.RGBA{
		Pix:    pixData,
		Stride: w * 4,
		Rect:   image.Rect(0, 0, w, h),
	}, nil
}

// DecodeAnimFrames 解码动画WebP的所有帧
func DecodeAnimFrames(data []byte) ([]*Frame, error) {
	if len(data) == 0 {
		return nil, errors.New("DecodeAnimFrames: data is empty")
	}
	
	var frames **C.uint8_t
	var timestamps, durations *C.int
	var frameCount, width, height C.int
	
	result := C.webpDecodeAnimFrames(
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		&frames, &timestamps, &durations,
		&frameCount, &width, &height,
	)
	
	if result == 0 {
		return nil, errors.New("DecodeAnimFrames: failed to decode frames")
	}
	
	defer C.webpFreeFrames(frames, timestamps, durations, frameCount)
	
	w, h := int(width), int(height)
	pixelCount := w * h * 4
	goFrames := make([]*Frame, int(result))
	
	// 转换C数组到Go切片
	frameSlice := (*[1 << 20]*C.uint8_t)(unsafe.Pointer(frames))[:int(result):int(result)]
	timestampSlice := (*[1 << 20]C.int)(unsafe.Pointer(timestamps))[:int(result):int(result)]
	durationSlice := (*[1 << 20]C.int)(unsafe.Pointer(durations))[:int(result):int(result)]
	
	for i := 0; i < int(result); i++ {
		if frameSlice[i] != nil {
			pixData := make([]byte, pixelCount)
			copy(pixData, (*[1 << 30]byte)(unsafe.Pointer(frameSlice[i]))[:pixelCount:pixelCount])
			
			goFrames[i] = &Frame{
				Image: &image.RGBA{
					Pix:    pixData,
					Stride: w * 4,
					Rect:   image.Rect(0, 0, w, h),
				},
				Timestamp: int(timestampSlice[i]),
				Duration:  int(durationSlice[i]),
			}
		}
	}
	
	return goFrames, nil
}

// ConvertAnimToStatic 将动画WebP转换为静态WebP（提取第一帧）
func ConvertAnimToStatic(data []byte, quality float32) ([]byte, error) {
	// 首先检查是否为动画
	isAnim, err := IsAnimated(data)
	if err != nil {
		return nil, err
	}
	
	if !isAnim {
		// 如果不是动画，直接重新编码以应用质量设置
		img, err := DecodeRGBA(data)
		if err != nil {
			return nil, err
		}
		return EncodeRGBA(img, quality)
	}
	
	// 解码第一帧
	firstFrame, err := DecodeAnimFirstFrame(data)
	if err != nil {
		return nil, err
	}
	
	// 编码为静态WebP
	return EncodeRGBA(firstFrame, quality)
}