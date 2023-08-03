package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/google/uuid"
	"github.com/overmindtech/ovm-cli/tracing"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// getChangeCmd represents the get-change command
var getChangeCmd = &cobra.Command{
	Use:   "get-change {--uuid ID | --change https://app.overmind.tech/changes/c772d072-6b0b-4763-b7c5-ff5069beed4c}",
	Short: "Displays the contents of a change.",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `get-change` flags")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		exitcode := GetChange(sigs, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func GetChange(signals chan os.Signal, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}

	var changeUuid uuid.UUID
	if viper.GetString("uuid") != "" {
		changeUuid, err = uuid.Parse(viper.GetString("uuid"))
		if err != nil {
			log.Errorf("invalid --uuid value '%v', error: %v", viper.GetString("uuid"), err)
			return 1
		}
	}

	if viper.GetString("change") != "" {
		changeUrl, err := url.ParseRequestURI(viper.GetString("change"))
		if err != nil {
			log.Errorf("invalid --change value '%v', error: %v", viper.GetString("change"), err)
			return 1
		}
		changeUuid, err = uuid.Parse(path.Base(changeUrl.Path))
		if err != nil {
			log.Errorf("invalid --change value '%v', couldn't parse: %v", viper.GetString("change"), err)
			return 1
		}
	}

	if changeUuid == uuid.Nil {
		log.Error("no change specified; use one of --uuid or --change")
		return 1
	}

	ctx := context.Background()
	ctx, span := tracing.Tracer().Start(ctx, "CLI GetChange", trace.WithAttributes(
		attribute.String("om.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	ctx, err = ensureToken(ctx, signals)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(log.Fields{
			"url": viper.GetString("url"),
		}).Error("failed to authenticate")
		return 1
	}

	// apply a timeout to the main body of processing
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := AuthenticatedChangesClient(ctx)
	response, err := client.GetChange(ctx, &connect.Request[sdp.GetChangeRequest]{
		Msg: &sdp.GetChangeRequest{
			UUID: changeUuid[:],
		},
	})
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(log.Fields{
			"change-url": viper.GetString("change-url"),
		}).Error("failed to get change")
		return 1
	}
	log.WithContext(ctx).WithFields(log.Fields{
		"change-uuid":        uuid.UUID(response.Msg.Change.Metadata.UUID),
		"change-created":     response.Msg.Change.Metadata.CreatedAt.AsTime(),
		"change-name":        response.Msg.Change.Properties.Title,
		"change-description": response.Msg.Change.Properties.Description,
	}).Info("found change")

	switch viper.GetString("format") {
	case "json":
		b, _ := json.MarshalIndent(response.Msg.Change, "", "  ")
		fmt.Println(string(b))
	case "markdown":
		changeUrl := fmt.Sprintf("%v/changes/%v", viper.GetString("frontend"), changeUuid.String())
		if response.Msg.Change.Metadata.NumAffectedApps != 0 || response.Msg.Change.Metadata.NumAffectedItems != 0 {
			// we have affected stuff
			fmt.Printf(`## Blast Radius  &nbsp; ·  &nbsp; [View in Overmind](%v) <img align="center" width="16" src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADAAAAAwCAYAAABXAvmHAAAACXBIWXMAABYlAAAWJQFJUiTwAAAAAXNSR0IArs4c6QAAAARnQU1BAACxjwv8YQUAAAPwSURBVHgB7VrdVdswGFVCBkg3cDdIJyBMUHyAZ5INyASlE6SdAPPKX+gENRPABmSDZgBIem9ipZ8VKZaR7fRwuOfk2LJk+V79fLoSKPWB3aKl/gNMJpMuLvzN4jielXl3ZwJAuo/L6Xw+5zUSWRSQ4vfr6OgoKaqncQEg3lssFmP8+h7Fpyg3Oj4+vncVaKsGcXd3xxb/7UmeiFqtFjRPvrkKNCaA5EE8UauxrpFA0EG73f6E4dLilWmUu5Tv4tm5S0QjQ0iQ15iCLOZr/OR6B4Qj9pYS8wPvHOCdVJarvQcc5A+2kSeQvyzH8voZBF2Y5WoVsIX81Of9TEQsHkU3NzeHskxtAkLJa7CnjDkxkPm1CKiKvIasC1FpX+ZVLqBq8kSn05HzpZut3EtUKqAO8oTFXqwFVBZG6yJPsMURgf7oNNcMfV9JD/iQl91eFi8vLz1Zt8wLFsAFB+R/yA+Y5BH6HtmCt7e3tAWRKgnUdyqSaS5PBQLExurfmLS1PP2MbsFDlH9kj3lWrzLBA51G/TmbESSAzpKkROUbY55pkP4uHnU53HxECDuxBN57qtRKoMKBSCauCXtycnIOcZ+VGL8cdtuGk8ULzfb29mKzXJAAfGC9qJhda8LibdgTF7ayNiMHjGwNFCQAhCKRfCoqb3ob7guyndkaDvJD1+4sdAh1BTmvvazpbV5fX3PmrAx5wksAJxxC4DN+Y/kc0WVNukycN9aMfVEHg0Ik8oZF++JCAWKRivA7kxMPrTUVRXvKE9LboBEifZ/1zk+1midDn019Z1umbYWVEwnj+QH5S+IQw7CYKg9wuKE3dTLXc9jAn+Fypjzh7AGXPZBlMH7lacHg6urKqxeM4VbqHMiEVYCvMUN8TzEEUp1GnPayCtLb4DtTFYANAZm3ScSjra4SAoZiMi9DYJEIw9s8qABsCJBLt/KwxMzj4ZN4RBHPrmMQ09ug1xIVgNx+4Pr6ug/CawFc/n39PCblAD0xlmsDz3nk+mBZpO4RaWIVgFwPGF2blNmMMORBwBeSYppzo4D8DN8bqUCYYbSvb4q8jQ2Z4JhRpoA8Mapit2bOgUjcF3obFzzIey1SPmjLD7lIvBV1kydkD8yMj795D5u9Xzt5Yi2ALS7NmSrhbUw0RZ7IzQGEwFTfZ96mNJokT5gCcmeQvt5Go2nyxMbBFn2/IOB9OLUL8oTNSgxF0svbcCOyC/KE9WgRloKnCKaXSbLFjUcbsyxK9bK5MjDKNkKecJ6NOkQUgVFs1BR5YuvhLv8aQoOm8kPDXhG8D611FfagDLxOp+k0cfmqVl5JLnCc5Ck2KJfc3KgdoPTxesi/BXzgPeIvmnKmx43NP2kAAAAASUVORK5CYII=" alt="chain link icon" />

> **Warning**
> Overmind identified potentially affected apps and items as a result of this pull request.

<br>

| <img align="center" width="16" src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAEAAAABACAYAAACqaXHeAAAACXBIWXMAABYlAAAWJQFJUiTwAAAAAXNSR0IArs4c6QAAAARnQU1BAACxjwv8YQUAAAM0SURBVHgB7ZqxUhsxEIb/FZg0eQB6eAB6aJwUYVKEZMBFKmjyAFR5hlRp0qUxbQ4GSJEJk0zcmN4PEHo/ABUM2uyeQwY7htPZWs4H+mZc+Kyx7n6tpP1XByQSiUQikXisEErSed1aAdEqvF9jYBEGyE39hn4WaK+ZZX0YEixAp9V6igu/zaAt3CME3seCUyHOYUCQAPrw/oI/SuNlVAANomHXQgQX1EpGvqqHV1j7lnuAAYUCdF62Fu877Meh95CvP5EpjoAGmSg/CR5YQ2QKBfDwlYX+fzCvIjKFAlQ590chg213HoYQ49w7OhCVe2igb72nT4KJADJSfUmWPjSPsx5mnOgCMNEBNdC2SlxiE1UAefi950dZGzUimgA68uMefta9QxQBdM5r2N+89s87MG/J9gVLeLBTSbbI652NzVLeISwVLrwDGurw2jtUkUHmfUrf+QAEMLUAOvrNr9n3oYs18g5TC+CJTm9+r5t3mFoA+YPu0IWaeYfp14BGvgLf6LRe3mFqAUZX27p5hzi7QI1JAuCRY2qHy0Lkeh585vxgYfUOi475hVUKrcyEAFo3wJzY58Mv3TE/tzuvWuvSaNtCiMqnQF47eELvmodZ97Y2eaZ5SbusQkWm+jWAwxxc81vWdxIliEylAoz1EXegUSLTJWqVqVIBxEecoCRSYzxDRAoFYJ2jRjCV/2/2w6n3tBRHwIjbe2i4gAZdGCHzubRvIBd3KywUoHmU9bTeBwOc5yWURRIjRCRoEXRS7/tbeIyKrAErnTet4PO+n5IQxT4dChIgt7xyPm8RCf6K30v9rvChtNJEkg0iMsHboIogZe9PdElvpdx0Qs5FiQgi5AVUHd3b2vyQKPHz/NnibLD0O0JF/NrY6mBCeHCkdiJmaLA9ztESrvyyThVMyLPj/eZdv8+WG9QRZjE918Pi2WCIhkkFETxykgCIjKV3KEtI7hI/AmbIO2h5rahNdAEsvUNZ3IJrF7ZBZCy9Qxn0HkIqTSaLoJV3CEX7diPvK9zR1ob8HYFL7BDzJu4RHXlX4h0l4zxrYGJ8AzvO0RJ7m4NTgut74lNdf3QKIpFIJBKJRCKAPxZBM7U9oOuSAAAAAElFTkSuQmCC" alt="icon for blast radius items" /> &nbsp;Affected items |
| ------------- |
| [%v items](%v) |
`, changeUrl, response.Msg.Change.Metadata.NumAffectedItems, changeUrl)
		} else {
			fmt.Printf(`## Blast Radius  &nbsp; ·  &nbsp; [View in Overmind](%v) <img align="center" width="16" src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADAAAAAwCAYAAABXAvmHAAAACXBIWXMAABYlAAAWJQFJUiTwAAAAAXNSR0IArs4c6QAAAARnQU1BAACxjwv8YQUAAAPwSURBVHgB7VrdVdswGFVCBkg3cDdIJyBMUHyAZ5INyASlE6SdAPPKX+gENRPABmSDZgBIem9ipZ8VKZaR7fRwuOfk2LJk+V79fLoSKPWB3aKl/gNMJpMuLvzN4jielXl3ZwJAuo/L6Xw+5zUSWRSQ4vfr6OgoKaqncQEg3lssFmP8+h7Fpyg3Oj4+vncVaKsGcXd3xxb/7UmeiFqtFjRPvrkKNCaA5EE8UauxrpFA0EG73f6E4dLilWmUu5Tv4tm5S0QjQ0iQ15iCLOZr/OR6B4Qj9pYS8wPvHOCdVJarvQcc5A+2kSeQvyzH8voZBF2Y5WoVsIX81Of9TEQsHkU3NzeHskxtAkLJa7CnjDkxkPm1CKiKvIasC1FpX+ZVLqBq8kSn05HzpZut3EtUKqAO8oTFXqwFVBZG6yJPsMURgf7oNNcMfV9JD/iQl91eFi8vLz1Zt8wLFsAFB+R/yA+Y5BH6HtmCt7e3tAWRKgnUdyqSaS5PBQLExurfmLS1PP2MbsFDlH9kj3lWrzLBA51G/TmbESSAzpKkROUbY55pkP4uHnU53HxECDuxBN57qtRKoMKBSCauCXtycnIOcZ+VGL8cdtuGk8ULzfb29mKzXJAAfGC9qJhda8LibdgTF7ayNiMHjGwNFCQAhCKRfCoqb3ob7guyndkaDvJD1+4sdAh1BTmvvazpbV5fX3PmrAx5wksAJxxC4DN+Y/kc0WVNukycN9aMfVEHg0Ik8oZF++JCAWKRivA7kxMPrTUVRXvKE9LboBEifZ/1zk+1midDn019Z1umbYWVEwnj+QH5S+IQw7CYKg9wuKE3dTLXc9jAn+Fypjzh7AGXPZBlMH7lacHg6urKqxeM4VbqHMiEVYCvMUN8TzEEUp1GnPayCtLb4DtTFYANAZm3ScSjra4SAoZiMi9DYJEIw9s8qABsCJBLt/KwxMzj4ZN4RBHPrmMQ09ug1xIVgNx+4Pr6ug/CawFc/n39PCblAD0xlmsDz3nk+mBZpO4RaWIVgFwPGF2blNmMMORBwBeSYppzo4D8DN8bqUCYYbSvb4q8jQ2Z4JhRpoA8Mapit2bOgUjcF3obFzzIey1SPmjLD7lIvBV1kydkD8yMj795D5u9Xzt5Yi2ALS7NmSrhbUw0RZ7IzQGEwFTfZ96mNJokT5gCcmeQvt5Go2nyxMbBFn2/IOB9OLUL8oTNSgxF0svbcCOyC/KE9WgRloKnCKaXSbLFjUcbsyxK9bK5MjDKNkKecJ6NOkQUgVFs1BR5YuvhLv8aQoOm8kPDXhG8D611FfagDLxOp+k0cfmqVl5JLnCc5Ck2KJfc3KgdoPTxesi/BXzgPeIvmnKmx43NP2kAAAAASUVORK5CYII=" alt="chain link icon" />

> **✅ Checks complete**
> Overmind didn't identify any potentially affected apps and items as a result of this pull request.

`, changeUrl)
		}
	}

	return 0
}

func init() {
	rootCmd.AddCommand(getChangeCmd)

	getChangeCmd.PersistentFlags().String("change", "", "The frontend URL of the change to get")
	getChangeCmd.PersistentFlags().String("uuid", "", "The UUID of the change that should be displayed.")
	getChangeCmd.MarkFlagsMutuallyExclusive("change", "uuid")

	getChangeCmd.PersistentFlags().String("frontend", "https://app.overmind.tech/", "The frontend base URL")
	getChangeCmd.PersistentFlags().String("format", "json", "How to render the change. Possible values: json, markdown")

	getChangeCmd.PersistentFlags().String("timeout", "1m", "How long to wait for responses")
}
