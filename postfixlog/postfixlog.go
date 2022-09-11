package postfixlog

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"time"
	"unsafe"
)

// StatsBin :
type StatsBin struct {
	delays            sort.Float64Slice
	receivingDelay    sort.Float64Slice
	queuingDelay      sort.Float64Slice
	connectionDelay   sort.Float64Slice
	transmissionDelay sort.Float64Slice
	c2xx              float64
	c4xx              float64
	c5xx              float64
	total             float64
}

// Stats :
type Stats struct {
	delays            float64
	receivingDelay    float64
	queuingDelay      float64
	connectionDelay   float64
	transmissionDelay float64
	dsn               int
}

func bFloat64(b []byte) float64 {
	f, _ := strconv.ParseFloat(*(*string)(unsafe.Pointer(&b)), 64)
	return f
}

func bInt(b []byte) int {
	i, _ := strconv.Atoi(*(*string)(unsafe.Pointer(&b)))
	return i
}

func round(f float64) int64 {
	return int64(math.Round(f)) - 1
}

// NewStatsBin :
func NewStatsBin() *StatsBin {
	return &StatsBin{}
}

// Append :
func (sb *StatsBin) Append(s *Stats) {
	switch s.dsn {
	case 2:
		sb.c2xx++
	case 4:
		sb.c4xx++
	case 5:
		sb.c5xx++
	}
	sb.total++

	sb.delays = append(sb.delays, s.delays)
	sb.receivingDelay = append(sb.receivingDelay, s.receivingDelay)
	sb.queuingDelay = append(sb.queuingDelay, s.queuingDelay)
	sb.connectionDelay = append(sb.connectionDelay, s.connectionDelay)
	sb.transmissionDelay = append(sb.transmissionDelay, s.transmissionDelay)
}

// DisplayDelay :
func (sb *StatsBin) DisplayDelay(now uint64, key string, f64s sort.Float64Slice) {
	sort.Sort(f64s)
	fl := float64(len(f64s))
	sum := float64(0)
	for _, x := range f64s {
		sum += x
	}

	if fl > 0 {
		fmt.Printf("postfixlog.%s_delay.average\t%f\t%d\n", key, sum/fl, now)
		fmt.Printf("postfixlog.%s_delay.99_percentile\t%f\t%d\n", key, f64s[round(fl*0.99)], now)
		fmt.Printf("postfixlog.%s_delay.95_percentile\t%f\t%d\n", key, f64s[round(fl*0.95)], now)
		fmt.Printf("postfixlog.%s_delay.90_percentile\t%f\t%d\n", key, f64s[round(fl*0.90)], now)
	}
}

// Display :
func (sb *StatsBin) Display(duration float64) {
	now := uint64(time.Now().Unix())
	sb.DisplayDelay(now, "total", sb.delays)
	sb.DisplayDelay(now, "recving", sb.receivingDelay)
	sb.DisplayDelay(now, "queuing", sb.queuingDelay)
	sb.DisplayDelay(now, "connection", sb.connectionDelay)
	sb.DisplayDelay(now, "transmission", sb.transmissionDelay)

	if duration > 0 {
		fmt.Printf("postfixlog.transfer_num.2xx_count\t%f\t%d\n", sb.c2xx/duration, now)
		fmt.Printf("postfixlog.transfer_num.4xx_count\t%f\t%d\n", sb.c4xx/duration, now)
		fmt.Printf("postfixlog.transfer_num.5xx_count\t%f\t%d\n", sb.c5xx/duration, now)
		fmt.Printf("postfixlog.transfer_total.count\t%f\t%d\n", sb.total/duration, now)
	}
	if sb.total > 0 {
		fmt.Printf("postfixlog.transfer_ratio.2xx_percentage\t%f\t%d\n", sb.c2xx*100/sb.total, now)
		fmt.Printf("postfixlog.transfer_ratio.4xx_percentage\t%f\t%d\n", sb.c4xx*100/sb.total, now)
		fmt.Printf("postfixlog.transfer_ratio.5xx_percentage\t%f\t%d\n", sb.c5xx*100/sb.total, now)
	}
}

// Apr 19 12:50:52 relaymail1 postfix/smtp[7570]: 69FFFC00B6: to=<xxxxxxx@example.jp>, relay=x.x.x.x[y.y.y.y]:25, delay=0.31, delays=0.04/0/0.09/0.17, dsn=2.0.0, status=sent (250 Ok)

var re = regexp.MustCompile(`, delay=(.+?), delays=(.+?)/(.+?)/(.+?)/(.+?), dsn=(\d)\.`)

// Parse :
func Parse(d1 []byte) (*Stats, error) {
	rs := re.FindSubmatch(d1)
	if len(rs) == 0 {
		return nil, fmt.Errorf("Not matched")
	}
	return &Stats{
		bFloat64(rs[1]),
		bFloat64(rs[2]),
		bFloat64(rs[3]),
		bFloat64(rs[4]),
		bFloat64(rs[5]),
		bInt(rs[6]),
	}, nil
}
