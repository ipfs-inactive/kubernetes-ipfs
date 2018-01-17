# What is trigger server

The trigger server's purpose is defined in ipfs/ipfs-cluster/issues/244. The goal is to create an endpoint
which can be triggered remotely and automated to run the tests, obtaining their results.


To build RPM/DEB
---

To build the RPM/DEBs in this folder, run the following commands from within this folder:


```
# Create FPM docker image with tar included (See PR https://github.com/jordansissel/fpm/pull/1455)
docker build -t "fpm" .
# Run RPM/DEB builds
./build.sh
