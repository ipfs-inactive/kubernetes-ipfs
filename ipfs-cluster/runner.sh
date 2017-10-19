#! /bin/bash
# Run all kubernetes tests with parameters

# Should be at least two arguments passed in
if [[ $# -lt "2" ]]; then
    echo "usage: ./runner.sh [Number of nodes] [Number of pins per test]"
    exit 1;
fi
# Need at least two nodes to make a cluster
if [[ $1 -lt "2" ]]; then
   echo "usage: ./runner.sh [Number of nodes] [Number of pins per test]"
   exit 1;
fi

# Need to make at least one pin
if [[ $2 -lt "1" ]]; then
   echo "usage: ./config-writer.sh [Number of nodes] [Number of pins per test]\n
   Need Y > 0 pins to run tests correctly"
   exit 1;
fi

go install ..

# Set the number of nodes in the deployment
NONBOOTSTRAP=$(( $1 - 1 ))
awk '
BEGIN {x=0}
{ if (match($0, /replicas:/)) {
    ++x;
    if (x==2) {
      printf("  replicas: ");
      print '$NONBOOTSTRAP'
    } else {
      print
    }
  } else {
    print
  }
} ' ipfs-cluster-deployment.yml > tmp-ipfs-cluster-deployment.yml
rm ipfs-cluster-deployment.yml
mv tmp-ipfs-cluster-deployment.yml ipfs-cluster-deployment.yml

./init.sh
./config-writer.sh $1 $2

if [[ $3 ]]; then
    exit 0
fi

FILE_NAMES=$(find tests -not -name "config.yml" -name "*.yml" | sort)

for file in $FILE_NAMES; do
  echo $file
  kubernetes-ipfs $file
done
