package bench

import (
	"github.com/nanxin/gadb"
	"os"
	"os/exec"
	"testing"
)

var deviceID = os.Getenv("DEVICE_ID")

func BenchmarkPullFB0UsingADBCLI(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("adb", "-s", deviceID, "pull", "/dev/graphics/fb0", "/dev/null")
		if err := cmd.Run(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPullFB0UsingClient(b *testing.B) {
	client, err := gadb.NewClient() // 使用默认配置
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	devices, err := client.DeviceList()
	for _, device := range devices {
		if device.Serial() == deviceID {
			outputFile, err := os.Create("output_file_name")
			if err != nil {
				return // 或者根据你的需求处理错误
			}
			defer outputFile.Close()
			err = device.Pull("/dev/graphics/fb0", outputFile)
			if err != nil {
				b.Errorf("Pull failed: %v", err)
			}

		}
	}
}
