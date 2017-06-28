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
NONBOOTSTRAP=`expr $1 - 1`
echo "51s/.*.*/  replicas: "$NONBOOTSTRAP"/" > sed-command.txt
sed -f sed-command.txt -i ipfs-cluster-deployment.yml
rm sed-command.txt
./init.sh
./config-writer.sh $1 $2

FILE_NAMES=("block_majority-5.yml"
            "block_minority-4.yml"
            "pin_and_unpin.yml"
            "pin_everywhere.yml"
            "pin_large_files-8.yml"
            "random_kill-13.yml"
            "random_reap-12.yml"
            "replication_factor.yml"
            "replication_self_heal-6.yml"
            "replication_update.yml"
            "start_and_check.yml"
            "sync_and_recover.yml"
            "add_rm_peers-10.yml"
            "add_rm_peers_rand_bootstrapper-11.yml"
            "add_rm_peers_pin-14.yml")

CLI_ARGS=(  ""
            ""
            ""
            ""
            ""
            ""
            ""
            ""
            ""
            ""
            ""
            ""
            ""
            ""
            "")
for i in "${!FILE_NAMES[@]}"; do
  kubernetes-ipfs "tests/"${FILE_NAMES[$i]}
  #" "${CLI_ARGS[$i]}
done
