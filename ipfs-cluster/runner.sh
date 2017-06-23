#! /bin/bash
# Run all kubernetes tests with parameters

# Should be at least one argument passed in
if [[ $# -lt "1" ]]; then
    exit 1;
fi

FILE_NAMES=("add_rm_peers-10.yml"
            "add_rm_peers_pin-14.yml"
            "add_rm_peers_rand_bootstrapper-11.yml"
            "block_majority-5.yml"
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
            "sync_and_recover.yml")

CLI_ARGS=(  "--param=N:"$1
            "--param=N:"$1
            "--param=N:"$1
            "--param=N:"$1
            "--param=N:"$1
            "--param=N:"$1
            "--param=N:"$1
            "--param=N:"$1
            "--param=N:"$1
            "--param=N:"$1
            "--param=N:"$1
            "--param=N:"$1
            "--param=N:"$1
            "--param=N:"$1
            "--param=N:"$1)
for i in "${!FILE_NAMES[@]}"; do
  kubernetes-ipfs "tests/"${FILE_NAMES[$i]}" "${CLI_ARGS[$i]}
done
