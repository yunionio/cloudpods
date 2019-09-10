package shell

import (
	"context"
	"fmt"

	"yunion.io/x/onecloud/pkg/util/printutils"
	"yunion.io/x/onecloud/pkg/util/redfish"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type IndicatorLEDOptions struct {
	}
	shellutils.R(&IndicatorLEDOptions{}, "led-get", "Get status of Indicator LED", func(cli redfish.IRedfishDriver, args *IndicatorLEDOptions) error {
		on, err := cli.GetIndicatorLED(context.Background())
		if err != nil {
			return err
		}
		fmt.Println("IndicatorLED", on)
		return nil
	})
	shellutils.R(&IndicatorLEDOptions{}, "led-on", "Set status of Indicator LED on", func(cli redfish.IRedfishDriver, args *IndicatorLEDOptions) error {
		err := cli.SetIndicatorLED(context.Background(), true)
		if err != nil {
			return err
		}
		fmt.Println("Success!")
		return nil
	})
	shellutils.R(&IndicatorLEDOptions{}, "led-off", "Set status of Indicator LED off", func(cli redfish.IRedfishDriver, args *IndicatorLEDOptions) error {
		err := cli.SetIndicatorLED(context.Background(), false)
		if err != nil {
			return err
		}
		fmt.Println("Success!")
		return nil
	})

	shellutils.R(&IndicatorLEDOptions{}, "power-get", "Get current power consumption", func(cli redfish.IRedfishDriver, args *IndicatorLEDOptions) error {
		powers, err := cli.GetPower(context.Background())
		if err != nil {
			return err
		}
		printutils.PrintInterfaceList(powers, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&IndicatorLEDOptions{}, "thermal-get", "Get current temperatures", func(cli redfish.IRedfishDriver, args *IndicatorLEDOptions) error {
		temps, err := cli.GetThermal(context.Background())
		if err != nil {
			return err
		}
		printutils.PrintInterfaceList(temps, 0, 0, 0, nil)
		return nil
	})
}
