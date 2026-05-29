from SCons.Script import COMMAND_LINE_TARGETS, Exit

Import("env")

UPLOAD_TARGETS = {"upload", "uploadfs", "uploadfsota", "uploadota"}
requested_targets = {str(target).lower() for target in COMMAND_LINE_TARGETS}

if requested_targets.intersection(UPLOAD_TARGETS):
    print(
        "A21 raw PlatformIO upload is forbidden. "
        "Run a21 firmware-upload-check for a dry-run receipt; "
        "real flashing requires a future explicit guarded A21 flash command."
    )
    Exit(1)
