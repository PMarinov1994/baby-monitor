### Dependencies
- libasound2-dev
- libopus-dev
- libopusfile-dev
- rpicam-apps


## Development dependencies
### node version manager (fnm)
- golang >=1.24
- curl -fsSL https://fnm.vercel.app/install | bash

## Performance tests

### mmal h264 hardware encoding on RpiZero2 32bit

```
params.BitRate          = 5_000_000
params.KeyFrameInterval = 30
```

| Resolution | Clean (fps) | Drawing (fps) |
|----------  |----------   |----------     |
| 1280x720   | 30    | 28      |
| 1920x1080  | 18    | N/A     |

### regular h264 software encoding on RpiZero2 64bit

```
params.BitRate          = 5_000_000
params.KeyFrameInterval = 30
```

| Resolution | Clean (fps) | Drawing (fps) |
|----------  |----------   |----------     |
| 1280x720   | 30    | 28      |
| 1920x1080  | 18    | N/A     |
