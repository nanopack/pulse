package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/nanopack/pulse/relay"
	"github.com/spf13/cobra"
)

func main() {
	host := ""
	id := ""
	collectors := []string{}
	cmd := &cobra.Command{
		Use:   "pulse-tester",
		Short: "insert dummy data to pulse",
		Run: func(ccmd *cobra.Command, args []string) {
			r, err := relay.NewRelay(host, id)
			if err != nil {
				fmt.Println(err)
				r.Close()
				return
			}

			for _, collector := range collectors {
				err = r.AddCollector(collector, nil, relay.NewPointCollector(randFunc))
				if err != nil {
					r.Close()
					return
				}
			}
			fmt.Println("started")
			<-time.After(10 * time.Hour)
		},
	}
	cmd.Flags().StringVar(&host, "host", "127.0.0.1:3000", "connection to pulse")
	cmd.Flags().StringVar(&id, "id", "computer1", "id of the host to insert into pulse")
	cmd.Flags().StringSliceVar(&collectors, "collectors", []string{"cpu_used", "ram_used"}, "fake data to insert into pulse")

	cmd.Execute()
}

func randFunc() float64 {
	fmt.Println("pulling data")
	return rand.Float64()
}
