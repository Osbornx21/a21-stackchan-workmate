# A21 Phase 5H Firmware Build Provenance Guard

Phase 5H tightens firmware release discipline beyond artifact naming and checksums.

Every packaged firmware candidate now records build provenance in both:

- `firmware/artifacts/a21-firmware-release-index.jsonl`
- the sibling `<artifact>.bin.manifest.json`

The required provenance is:

- `build_system`: `platformio`
- `platformio_env`: `a21_stackchan_cores3`
- `platformio_board`: `m5stack-cores3`
- `source_name`: `firmware.bin`
- `source_path`: a PlatformIO source binary under `.pio/build/a21_stackchan_cores3/firmware.bin`

Upload-path dry-run guards reject candidates when the release index and per-artifact manifest do not both contain matching build provenance. This means a manually renamed binary, a loose `firmware.bin`, a package built from the wrong PlatformIO environment, or an artifact produced from an X21/V21 path cannot pass `firmware-upload-check`, `firmware-device-check`, or `firmware-flash-plan`.

The guards also reject forbidden X21/V21 identities in the full artifact/checksum paths recorded inside the release index or per-artifact manifest. Matching filenames are not enough; the ledger path itself must remain A21-clean.

The package step itself now rejects the source PlatformIO `firmware.bin` if it does not already embed the expected A21 firmware ID, version, board, and git commit. This catches stale builds before they become release artifacts.

This phase still does not enable real flashing. It only makes future flashing safer by proving which A21 build environment produced the candidate binary.
