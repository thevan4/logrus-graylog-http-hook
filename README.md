# logrus-graylog-http-hook

Warning: default http client => insecure tls!

## Example

```golang
import (
	"github.com/sirupsen/logrus"
	graylogHttpHook "github.com/thevan4/logrus-graylog-http-hook"
)
...
	logrusLog := logrus.New()
	graylogHook := graylogHttpHook.NewGraylogHook(logger.Graylog.Address, logger.Graylog.Retries, logger.Graylog.Extra, nil)
	logrusLog.AddHook(graylogHook)
	logrusLog.SetOutput(ioutil.Discard)
```
