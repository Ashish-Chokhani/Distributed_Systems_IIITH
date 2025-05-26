#!/bin/bash

if ls ../../P2-BouncingBalls/*.py &>/dev/null; then
    executable="python3 ../../P2-BouncingBalls/*.py"
elif ls ../../P2-BouncingBalls/*.cpp &>/dev/null; then
    mpic++ -std=c++17 -o P2-BouncingBalls ../../P2-BouncingBalls/*.cpp
    executable="./P2-BouncingBalls"
elif ls ../../P2-BouncingBalls/*.c &>/dev/null; then
    mpicc -o P2-BouncingBalls ../../P2-BouncingBalls/*.c
    executable="./P2-BouncingBalls"
else
    echo "No Python, C, or C++ file found in ../../P2-BouncingBalls/"
    exit 1
fi

normalize_spaces() {
    sed -e 's/[[:space:]]\+/ /g' -e 's/[[:space:]]*$//' -e '/^$/d' "$1" > "$2"
}

mkdir -p results

total_marks=0

num_test_cases=$(ls testcases/*.in | wc -l)

for i in $(seq 1 $num_test_cases); do
    test_file="testcases/${i}.in"
    expected_output="testcases/${i}.out"

    all_passed=true

    for np in {1..12}; do
        mpiexec -np $np --use-hwthread-cpus --oversubscribe $executable $test_file results/2_${np}_${i}.txt

        normalize_spaces results/2_${np}_${i}.txt results/2_${np}_${i}_normalized.txt
        normalize_spaces $expected_output results/expected_${i}_normalized.txt

        if ! diff -q results/2_${np}_${i}_normalized.txt results/expected_${i}_normalized.txt > /dev/null; then
            all_passed=false
            break
        fi
    done


    if [ "$all_passed" = true ]; then
        printf "Test case $i: \e[32mPASSED\e[0m\n"
        marks=$(grep "^${i} " marks.txt | cut -d ' ' -f 2)
        total_marks=$(echo "$total_marks + $marks" | bc)
    else
        printf "Test case $i: \e[31mFAILED\e[0m\n"
    fi
done

echo -e "Final Score: $total_marks/30"

rm -rf 2 results/
