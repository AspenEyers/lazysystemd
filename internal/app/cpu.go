package app

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type cpuSampleMsg struct {
	total uint64
	idle  uint64
	err   error
}

// getCPUSample reads /proc/stat and returns aggregate CPU total+idle ticks.
func (m *Model) getCPUSample() tea.Cmd {
	return func() tea.Msg {
		f, err := os.Open("/proc/stat")
		if err != nil {
			return cpuSampleMsg{err: fmt.Errorf("open /proc/stat: %w", err)}
		}
		defer f.Close()

		sc := bufio.NewScanner(f)
		if !sc.Scan() {
			if err := sc.Err(); err != nil {
				return cpuSampleMsg{err: fmt.Errorf("scan /proc/stat: %w", err)}
			}
			return cpuSampleMsg{err: fmt.Errorf("scan /proc/stat: empty")}
		}

		// First line: cpu  user nice system idle iowait irq softirq steal guest guest_nice
		fields := strings.Fields(sc.Text())
		if len(fields) < 5 || fields[0] != "cpu" {
			return cpuSampleMsg{err: fmt.Errorf("unexpected /proc/stat format")}
		}

		var nums []uint64
		for _, f := range fields[1:] {
			v, parseErr := strconv.ParseUint(f, 10, 64)
			if parseErr != nil {
				return cpuSampleMsg{err: fmt.Errorf("parse /proc/stat: %w", parseErr)}
			}
			nums = append(nums, v)
		}

		var total uint64
		for _, v := range nums {
			total += v
		}

		// idle + iowait (if present) count as idle time.
		idle := nums[3]
		if len(nums) > 4 {
			idle += nums[4]
		}

		return cpuSampleMsg{total: total, idle: idle}
	}
}

