#! /bin/sh

# Exit immediately on non-zero exit code.
set -e;

# Print evaluated commands to stdout.
set -x;

# Start the resizer.
resizer -b $BUCKET_NAME -a 0.0.0.0:2222 -e https://s3.fr-par.scw.cloud
