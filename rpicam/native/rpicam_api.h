#ifndef __RPICAM_API__H
#define __RPICAM_API__H

#ifdef __cplusplus
#include <cstddef>
#include <cstdint>
#else
#include <stddef.h>
#include <stdint.h>
#endif

#ifdef __cplusplus
extern "C" {
#endif

typedef void (*CameraOutputReadyCallback)(unsigned char *mem, size_t size);

struct CameraParams {
	uint8_t                   loglevel;
	uint32_t                  width;
	uint32_t                  height;
	uint32_t                  framerate;
	CameraOutputReadyCallback cb_yuv420;
	CameraOutputReadyCallback cb_h264;
};

int startCamera(struct CameraParams *params);


#ifdef __cplusplus
}
#endif

#endif // __RPICAM_API__H
