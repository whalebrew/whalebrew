# Change log

## 0.1.0 - 2017-03-04
### Added
* Ask permission before installing a package that needs to listen on ports, access environment variables, or access files/directories on the host.
* A `-f` flag to `whalebrew install` to force install a package over an existing file.

## 0.0.4 - 2017-02-07
### Added
* `whalebrew edit` command for editing packages

### Fixed
* Files being written as root

## 0.0.3 - 2017-01-30
### Fixed

* Permission errors when running `whalebrew list`

## 0.0.2 - 2017-01-28
### Added

* Support for mapping ports.

### Fixed

* `whalebrew list` when install path contains folders.

## 0.0.1 - 2017-01-26

Initial release.
