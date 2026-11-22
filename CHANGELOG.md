# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [v0.2.1]

### Fixed

- `-geography` over a map with units: the level beside the folder is laid on a board without
  them, their kinds being the game's, as the level file is. It was refused with "no kind is
  defined".

## [v0.2.0]

### Added

- `-geography <folder>` writes the map a second time as files in metres, gram's
  `board/mapfile` (the heights as an image, the land's kinds as areas, the rivers, the roads,
  their bridges, fords and tunnels as lines, the fords and the places as points), and beside
  it `<folder>.yaml`, the level a board lays by it, laid on a board before the run ends. A map
  written so is planned a step of its heights clear of every limit.

### Changed

- gram v0.26.0.

## [v0.1.0]

gramap in a repository of its own, as it stood in gram-madeira-square's `cmd/gramap`: the maps
it makes are the same to the byte.

### Changed

- The command is the module's root: `go run github.com/kjkrol/gramap@v0.1.0`.
- Madeira's recipe is `testdata/madeira.yaml`, the recipe the tests hold the tool to; a game
  keeps its own.
