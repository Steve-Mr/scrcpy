# scrcpy-tui

`scrcpy-tui` is a standalone Terminal User Interface (TUI) helper application for `scrcpy`. It provides a dashboard to select devices, configure options, and launch `scrcpy` with presets.

## Features

- **Device Selection**: List and select connected Android devices via `adb`.
- **App Listing**: List installed applications from the device and select one to start.
- **Common Options**: Interactive dashboard to set resolution, virtual display (`--new-display`), FPS, bitrate, and codec.
- **Presets**: Define and use configuration presets via a YAML file.
- **Command Preview**: Real-time preview of the `scrcpy` command being generated.
- **One-click Run**: Execute the generated command directly from the TUI.
- **Portable**: Static binary with minimal dependencies, compatible with Linux 4.19+.

## Build Instructions

You need [Go](https://go.dev/doc/install) installed to build the tool.

```bash
cd devtools/scrcpy-tui
go build -o scrcpy-tui
```

To build for a specific architecture (e.g., Linux x64):

```bash
GOOS=linux GOARCH=amd64 go build -o scrcpy-tui
```

## Usage

Run the binary from the terminal:

```bash
./scrcpy-tui
```

### Controls

- **Arrows (Up/Down)**: Move the cursor or navigate lists.
- **Enter**: Select an item, edit a value, or run the command.
- **Esc**: Go back or cancel input.
- **q / Ctrl+C**: Quit the application.

### Configuration

You can define presets in a `config.yaml` file located in the same directory as the executable.

Example `config.yaml`:

```yaml
presets:
  - name: "High Quality (1080p, 60fps)"
    max_resolution: "1920"
    max_fps: 60
    video_codec: "h265"
    video_bit_rate: "16M"
  - name: "Low Latency (720p, 30fps)"
    max_resolution: "1280"
    max_fps: 30
    video_codec: "h264"
    video_bit_rate: "4M"
  - name: "VLC Player"
    max_resolution: "1920"
    max_fps: 60
    args: ["--no-audio"]
```

## Binary Detection

The tool automatically looks for `scrcpy` and `adb` in:
1. The current working directory.
2. The system `PATH`.

You can use the "Add current dir to PATH" option within the dashboard to simplify usage if you have the binaries in the same folder.
