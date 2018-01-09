if [ -z $1 ]
then
	printf "%s: missing version\n" $0
	printf "usage: %s <version>\n" $0
	printf "example: %s 1.0.0\n" $0
	exit 1
fi
VERSION=$1
if [[ ! $(docker images | grep fpm) ]]
then
	docker build -t="fpm" github.com/jordansissel/fpm/
fi
go build triggerserver.go
docker run -v $(pwd):/data --rm fpm /bin/sh -c "cd /data && fpm -s dir -t rpm -n triggerserver -v $VERSION -p /data/ --deb-no-default-config-files triggerserver=/usr/bin/ triggerserver.service=/usr/lib/systemd/system/"
docker run -v $(pwd):/data --rm fpm /bin/sh -c "cd /data && fpm -s dir -t deb -n triggerserver -v $VERSION -p /data/ --deb-no-default-config-files triggerserver=/usr/bin/ triggerserver.service=/lib/systemd/system/"
rm triggerserver
