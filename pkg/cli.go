package pkg

import (
	"adb-kit-go/pkg/adb"
	"fmt"
	"github.com/spf13/cobra"
	"log"
)

func main() {
	var rootCmd = &cobra.Command{Use: "adbcli"}

	var pubkeyConvertCmd = &cobra.Command{
		Use:   "pubkey-convert <file>",
		Short: "Converts an ADB-generated public key into PEM format.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			file := args[0]
			format, _ := cmd.Flags().GetString("format")
			key, err := adb.ParsePublicKey([]byte(file))
			if err != nil {
				log.Fatalf("解析公钥失败: %v", err)
			}

			switch format {
			case "pem":
				fmt.Println(adb.PublicKeyToPem(key))
			case "openssh":
				fmt.Println(adb.PublicKeyToOpenSSH(key, "adbkey"))
			default:
				log.Fatalf("不支持的格式 '%s'", format)
			}
		},
	}
	pubkeyConvertCmd.Flags().StringP("format", "f", "pem", "format (pem or openssh)")

	var pubkeyFingerprintCmd = &cobra.Command{
		Use:   "pubkey-fingerprint <file>",
		Short: "Outputs the fingerprint of an ADB-generated public key.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			file := args[0]
			key, err := adb.ParsePublicKey([]byte(file))
			if err != nil {
				log.Fatalf("解析公钥失败: %v", err)
			}
			fmt.Printf("%s %s\n", key.Fingerprint, key.Comment)
		},
	}

	rootCmd.AddCommand(pubkeyConvertCmd, pubkeyFingerprintCmd)
	rootCmd.Execute()
}
