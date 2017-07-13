# Should be at least two arguments passed in
if [[ $# -lt "2" ]]; then
    echo "usage: ./config-writer.sh [Number of nodes] [Number of pins per test]"
    exit 1;
fi
# Need at least two nodes to make a cluster
if [[ $1 -lt "2" ]]; then
    echo "usage: ./config-writer.sh [Number of nodes] [Number of pins per test]\n
          Need N >= 2 nodes to make a cluster"
    exit 1;
fi

# Need to make at least one pin
if [[ $2 -lt "1" ]]; then
    echo "usage: ./config-writer.sh [Number of nodes] [Number of pins per test]\n
          Need Y > 0 pins to run tests correctly"
    exit 1;
fi

echo "params:" > "tests/config.yml"
echo "  N: "$1 >> "tests/config.yml"
echo "  Y: "$2 >> "tests/config.yml"
echo "  N_plus_1: ""$(( $1 + 1 ))" >> "tests/config.yml"
echo "  N_times_4_plus_2: ""$(( $1 * 4 + 2 ))" >> "tests/config.yml"
echo "  N_times_4: ""$(( $1 * 4 ))" >> "tests/config.yml"
echo "  N_times_10: ""$(( $1 * 10 ))" >> "tests/config.yml"
echo "  N_times_5: ""$(( $1 * 5 ))" >> "tests/config.yml"
echo "  N_minus_1: ""$(( $1 - 1 ))" >> "tests/config.yml"
echo "  N_minus_2: ""$(( $1 - 2 ))" >> "tests/config.yml"
echo "  N_minus_3: ""$(( $1 - 3 ))" >> "tests/config.yml"
echo "  Block_successes: ""$(( ($2 * 3) + ($2 * $1 * 2) ))" >> "tests/config.yml"
depart=$(( ($1 / 2) + 1 ))
stay=$(( $1 - $depart ))
add_rm=$(( 2 * ($1 * $1) + $depart + ($depart * $stay) ))
echo "  Add_Pin_Rm_success: ""$(( $add_rm + 3 * $1 * $2 ))" >> "tests/config.yml"
echo "  Add_Rm_success: "$add_rm >> "tests/config.yml"
