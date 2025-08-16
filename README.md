### Setup raspberry pi as serial gadget

Create symlincs at
<ROOTFS>/etc/systemd/system/getty.target.wants
```
mkdir -p /etc/systemd/system/getty.target.wants
cd /etc/systemd/system/getty.target.wants

sudo ln -s /lib/systemd/system/getty@.service getty@tty1.service
sudo ln -s /lib/systemd/system/getty@.service getty@ttyGS0.service
```

### Dependencies
- libasound2-dev
- libopus-dev
- libopusfile-dev
- rpicam-apps

```
sudo apt update && sudo apt upgrade -y && sudo apt install -y libasound2-dev libopus-dev libopusfile-dev rpicam-apps 
```

### Setup Codec Zero
Get the codec zero profiles
```
git clone https://github.com/raspberrypi/Pi-Codec.git ~/git/Pi-Codec
```

/etc/rc.local
```
#!/bin/sh
#
# rc.local
#
# This script is executed at the end of each multiuser runlevel.
# Make sure that the script will "exit 0" on success or any other
# value on error.
#
# In order to enable or disable this script just change the execution
# bits.
#
# By default this script does nothing.

sudo alsactl restore -f /home/pi/git/Pi-Codec/Codec_Zero_OnboardMIC_record_and_SPK_playback.state

exit 0
```

Create ~/.asoundrc
```
pcm.!default {
        type hw
        card Zero
}
```

## Development dependencies
- Common
```
sudo apt install -y xterm git vim
```

- golang >=1.24
```
wget https://go.dev/dl/go1.25.0.linux-arm64.tar.gz

sudo su
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.25.0.linux-arm64.tar.gz
exit

echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

go version
```

- Install Node >=22
```
curl -fsSL https://fnm.vercel.app/install | bash
source ~/.bashrc

fnm install 22
```

- Get the project
```
git clone https://github.com/PMarinov1994/baby-monitor.git
```

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
