#ifndef __RPICAM_API__H
#define __RPICAM_API__H

#ifdef __cplusplus
#include <cstddef>
#else
#include <stddef.h>
#endif

#ifdef __cplusplus
extern "C" {
#endif

typedef void (*CameraOutputReadyCallback)(unsigned char *mem, size_t size);

int startCamera(CameraOutputReadyCallback cb_info);


#ifdef __cplusplus
}
#endif

#endif // __RPICAM_API__H
