# Stash Enhanced

A fork of [Stash](https://github.com/stashapp/stash) with additional features.

Stash is a self-hosted webapp written in Go which organizes and serves your diverse content collection. Stash Enhanced is kept in sync with upstream Stash releases and adds its own enhancements on top. It is not an official Stash project.

## Enhancements

Features that are only available in Stash Enhanced:

- **Captions tab** on the scene details page, with clickable timecodes ([#2](https://github.com/808coder/stash-enhanced/pull/2))
- **File modification time column** in the Scene Duplicate Checker ([#4](https://github.com/808coder/stash-enhanced/pull/4))
- **File Details display mode** for the scene list ([#6](https://github.com/808coder/stash-enhanced/pull/6))
- **File Size filter** in the scene filter dialog ([#8](https://github.com/808coder/stash-enhanced/pull/8))

## Installing

There are no prebuilt binaries yet. A Docker image is planned. For now, build Stash Enhanced from source:

```sh
git clone https://github.com/808coder/stash-enhanced.git
cd stash-enhanced
make pre-ui
make generate
make ui
make build-release
```

See [DEVELOPMENT.md](docs/DEVELOPMENT.md) for prerequisites and platform-specific instructions.

### ffmpeg

Stash requires FFmpeg. If you don't have it installed, Stash will prompt you to download a copy during setup. It is recommended that Linux users install `ffmpeg` from their distro's package manager.

## Usage

Once the application is running, the interface is available (by default) at `http://localhost:9999`.

On first run, Stash will prompt you for some configuration options and media directories to index, called "Scanning" in Stash. After scanning, your media will be available for browsing, curating, editing, and tagging.

Stash Enhanced behaves like upstream Stash apart from its enhancements. For general usage, including scrapers, stash-box instances such as StashDB, and plugins, see the [upstream Stash documentation](https://docs.stashapp.cc).

## Contributing

Stash Enhanced is a personal fork. Issues and pull requests are welcome, but there is no guarantee that they will be addressed or merged.

Please only report bugs and request features here that relate to the enhancements of this fork. Everything else belongs to [upstream Stash](https://github.com/stashapp/stash/issues).

See [DEVELOPMENT.md](docs/DEVELOPMENT.md) for setting up a local development environment.

## License

Stash Enhanced is licensed under the [GNU Affero General Public License v3.0](LICENSE), like upstream Stash.
