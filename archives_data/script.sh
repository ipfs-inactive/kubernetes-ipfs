rm -r output
i=0
while IFS= read -r line
do
	# Skip comments
	if [ ${line:0:1} == '#' ]; then continue; fi
	num_files=$(echo $line | cut -d ',' -f 1 | awk '{$1=$1};1')
	min_file_size=$(echo $line | cut -d ',' -f 2 | awk '{$1=$1};1')
	max_file_size=$(echo $line | cut -d ',' -f 3 | awk '{$1=$1};1')
	nesting_depth=$(echo $line | cut -d ',' -f 4 | awk '{$1=$1};1')
	i=$((i+1))
	mkdir -p "output/line_$i"
	for ((j = 0; j < $num_files; j++))
	do
		dd if=/dev/zero of="output/line_$i/file_$j""_size_$max_file_size" bs=1 count=1 seek="$max_file_size" 2>/dev/null
		# Add a little entropy so they're not all the same hash
		head -c 25 /dev/urandom >> "output/line_$i/file_$j""_size_$max_file_size" 
	done
done < input.txt
echo "Complete."
