#ifndef __RPICAM_API__H
#define __RPICAM_API__H

#include <cstddef>

typedef void (*CameraOutputReadyCallback)(char *mem, size_t size);

#ifndef __cplusplus
extern "C"
#endif
int startCamera(CameraOutputReadyCallback cb_info);

#endif // __RPICAM_API__H
