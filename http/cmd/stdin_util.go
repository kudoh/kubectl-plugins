package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
)

func readNumFromStdIn(max int) (int, error) {
	fmt.Print("enter number > ")
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		n, err := strconv.ParseUint(s.Text(), 10, 32)
		if err != nil {
			fmt.Printf("[%s] is illegal format. enter number > ", s.Text())
			if int(n) > max {
				fmt.Printf("[%d] is unknown number. enter number > ", n)
			}
		} else {
			fmt.Println("You selected: ", n)
			return int(n), nil
		}
	}
	if err := s.Err(); err != nil {
		return 0, err
	}
	return 0, fmt.Errorf("unknown error")
}

func readStringFromStdIn(title string) (string, error) {

	fmt.Print("enter ", title, " > ")
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		return s.Text(), nil
	}
	if err := s.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("unknown error")
}
