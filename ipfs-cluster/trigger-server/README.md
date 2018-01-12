To build RPM/DEB
---

To build the RPM/DEBs in this folder, run the following commands from within this folder:


```
# Create FPM docker image with tar included (See PR https://github.com/jordansissel/fpm/pull/1455)
docker build -t "fpm" .
# Run RPM/DEB builds
./build.sh
