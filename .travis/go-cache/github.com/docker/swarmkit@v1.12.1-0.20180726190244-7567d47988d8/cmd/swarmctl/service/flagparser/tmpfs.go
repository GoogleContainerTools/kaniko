package flagparser

import (
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/docker/swarmkit/api"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

// parseTmpfs supports a simple tmpfs decl, similar to docker run.
//
// This should go away.
func parseTmpfs(flags *pflag.FlagSet, spec *api.ServiceSpec) error {
	if flags.Changed("tmpfs") {
		tmpfss, err := flags.GetStringSlice("tmpfs")
		if err != nil {
			return err
		}

		container := spec.Task.GetContainer()
		// TODO(stevvooe): Nasty inline parsing code, replace with mount syntax.
		for _, tmpfs := range tmpfss {
			parts := strings.SplitN(tmpfs, ":", 2)

			if len(parts) < 1 {
				return errors.Errorf("invalid mount spec: %v", tmpfs)
			}

			if len(parts[0]) == 0 || !path.IsAbs(parts[0]) {
				return errors.Errorf("invalid mount spec: %v", tmpfs)
			}

			m := api.Mount{
				Type:   api.MountTypeTmpfs,
				Target: parts[0],
			}

			if len(parts) == 2 {
				if strings.Contains(parts[1], ":") {
					// repeated colon is illegal
					return errors.Errorf("invalid mount spec: %v", tmpfs)
				}

				// BUG(stevvooe): Cobra stringslice actually doesn't correctly
				// handle comma separated values, so multiple flags aren't
				// really supported. We'll have to replace StringSlice with a
				// type that doesn't use the csv parser. This is good enough
				// for now.

				flags := strings.Split(parts[1], ",")
				var opts api.Mount_TmpfsOptions
				for _, flag := range flags {
					switch {
					case strings.HasPrefix(flag, "size="):
						meat := strings.TrimPrefix(flag, "size=")

						// try to parse this into bytes
						i, err := strconv.ParseInt(meat, 10, 64)
						if err != nil {
							// remove suffix and try again
							suffix := meat[len(meat)-1]
							meat = meat[:len(meat)-1]
							var multiplier int64 = 1
							switch suffix {
							case 'g':
								multiplier = 1 << 30
							case 'm':
								multiplier = 1 << 20
							case 'k':
								multiplier = 1 << 10
							default:
								return errors.Errorf("invalid size format: %v", flag)
							}

							// reparse the meat
							var err error
							i, err = strconv.ParseInt(meat, 10, 64)
							if err != nil {
								return err
							}

							i *= multiplier
						}
						opts.SizeBytes = i
					case strings.HasPrefix(flag, "mode="):
						meat := strings.TrimPrefix(flag, "mode=")
						i, err := strconv.ParseInt(meat, 8, 32)
						if err != nil {
							return err
						}
						opts.Mode = os.FileMode(i)
					case flag == "ro":
						m.ReadOnly = true
					case flag == "rw":
						m.ReadOnly = false
					default:
						return errors.New("unsupported flag")
					}
				}
				m.TmpfsOptions = &opts
			}

			container.Mounts = append(container.Mounts, m)
		}
	}

	return nil
}
