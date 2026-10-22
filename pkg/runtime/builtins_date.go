package runtime

import (
	"fmt"
	"strings"
	"time"
)

func newDateUnit() *UnitInstance {
	var dateUnit *UnitInstance

	dateUnit = &UnitInstance{
		UnitName:   "date",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"now": func(args []Value) (Value, error) {
				return dateToMap(time.Now()), nil
			},
			"Now": func(args []Value) (Value, error) {
				return dateToMap(time.Now()), nil
			},
			"iso": func(args []Value) (Value, error) {
				t := time.Now()
				if len(args) > 0 {
					var err error
					t, err = parseTimeToTime(args[0])
					if err != nil {
						return NilValue{}, err
					}
				}
				return StringValue{Val: t.Format(time.RFC3339)}, nil
			},
			"Iso": func(args []Value) (Value, error) {
				t := time.Now()
				if len(args) > 0 {
					var err error
					t, err = parseTimeToTime(args[0])
					if err != nil {
						return NilValue{}, err
					}
				}
				return StringValue{Val: t.Format(time.RFC3339)}, nil
			},
			"today": func(args []Value) (Value, error) {
				return StringValue{Val: time.Now().Format("2006-01-02")}, nil
			},
			"Today": func(args []Value) (Value, error) {
				return StringValue{Val: time.Now().Format("2006-01-02")}, nil
			},
			"timestamp": func(args []Value) (Value, error) {
				t := time.Now()
				if len(args) > 0 {
					var err error
					t, err = parseTimeToTime(args[0])
					if err != nil {
						return NilValue{}, err
					}
				}
				return NumberValue{Val: float64(t.UnixMilli())}, nil
			},
			"Timestamp": func(args []Value) (Value, error) {
				t := time.Now()
				if len(args) > 0 {
					var err error
					t, err = parseTimeToTime(args[0])
					if err != nil {
						return NilValue{}, err
					}
				}
				return NumberValue{Val: float64(t.UnixMilli())}, nil
			},
			"format": func(args []Value) (Value, error) {
				if len(args) == 0 {
					return StringValue{Val: time.Now().Format("2006-01-02 15:04:05")}, nil
				}
				if len(args) == 1 {
					// format("YYYY-MM-DD") sur now
					fmtStr := convertDateFormat(args[0].String())
					return StringValue{Val: time.Now().Format(fmtStr)}, nil
				}
				t, err := parseTimeToTime(args[0])
				if err != nil {
					return NilValue{}, err
				}
				fmtStr := convertDateFormat(args[1].String())
				return StringValue{Val: t.Format(fmtStr)}, nil
			},
			"Format": func(args []Value) (Value, error) {
				if len(args) == 0 {
					return StringValue{Val: time.Now().Format("2006-01-02 15:04:05")}, nil
				}
				if len(args) == 1 {
					fmtStr := convertDateFormat(args[0].String())
					return StringValue{Val: time.Now().Format(fmtStr)}, nil
				}
				t, err := parseTimeToTime(args[0])
				if err != nil {
					return NilValue{}, err
				}
				fmtStr := convertDateFormat(args[1].String())
				return StringValue{Val: t.Format(fmtStr)}, nil
			},
			"parse": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("date.parse requires date string argument")
				}
				str := args[0].String()
				if len(args) >= 2 {
					fmtStr := convertDateFormat(args[1].String())
					t, err := time.Parse(fmtStr, str)
					if err != nil {
						return NilValue{}, fmt.Errorf("date.parse failed: %w", err)
					}
					return dateToMap(t), nil
				}
				t, err := parseTimeToTime(StringValue{Val: str})
				if err != nil {
					return NilValue{}, err
				}
				return dateToMap(t), nil
			},
			"Parse": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("date.parse requires date string argument")
				}
				str := args[0].String()
				if len(args) >= 2 {
					fmtStr := convertDateFormat(args[1].String())
					t, err := time.Parse(fmtStr, str)
					if err != nil {
						return NilValue{}, fmt.Errorf("date.parse failed: %w", err)
					}
					return dateToMap(t), nil
				}
				t, err := parseTimeToTime(StringValue{Val: str})
				if err != nil {
					return NilValue{}, err
				}
				return dateToMap(t), nil
			},
			"add": func(args []Value) (Value, error) {
				if len(args) < 2 {
					return NilValue{}, fmt.Errorf("date.add requires at least 2 arguments (date, amount, [unit])")
				}
				t, err := parseTimeToTime(args[0])
				if err != nil {
					return NilValue{}, err
				}
				if len(args) == 2 {
					// date.add(date, "2h") ou date.add(date, 3600)
					if durVal, ok := args[1].(DurationValue); ok {
						return dateToMap(t.Add(durVal.Val)), nil
					}
					if sVal, ok := args[1].(StringValue); ok {
						d, err := time.ParseDuration(sVal.Val)
						if err == nil {
							return dateToMap(t.Add(d)), nil
						}
					}
					if nVal, ok := args[1].(NumberValue); ok {
						return dateToMap(t.Add(time.Duration(nVal.Val) * time.Millisecond)), nil
					}
				}
				if len(args) >= 3 {
					amount := 0.0
					if nVal, ok := args[1].(NumberValue); ok {
						amount = nVal.Val
					}
					unit := strings.ToLower(args[2].String())
					switch unit {
					case "ms", "millisecond", "milliseconds":
						t = t.Add(time.Duration(amount) * time.Millisecond)
					case "s", "sec", "second", "seconds":
						t = t.Add(time.Duration(amount) * time.Second)
					case "m", "min", "minute", "minutes":
						t = t.Add(time.Duration(amount) * time.Minute)
					case "h", "hour", "hours":
						t = t.Add(time.Duration(amount) * time.Hour)
					case "d", "day", "days":
						t = t.AddDate(0, 0, int(amount))
					case "w", "week", "weeks":
						t = t.AddDate(0, 0, int(amount*7))
					case "month", "months":
						t = t.AddDate(0, int(amount), 0)
					case "y", "year", "years":
						t = t.AddDate(int(amount), 0, 0)
					default:
						return NilValue{}, fmt.Errorf("date.add: unknown unit '%s'", unit)
					}
					return dateToMap(t), nil
				}
				return dateToMap(t), nil
			},
			"Add": func(args []Value) (Value, error) {
				return dateUnit.Methods["add"](args)
			},
			"diff": func(args []Value) (Value, error) {
				if len(args) < 2 {
					return NilValue{}, fmt.Errorf("date.diff requires 2 date arguments (dateA, dateB, [unit])")
				}
				tA, err := parseTimeToTime(args[0])
				if err != nil {
					return NilValue{}, err
				}
				tB, err := parseTimeToTime(args[1])
				if err != nil {
					return NilValue{}, err
				}
				diffDur := tA.Sub(tB)
				unit := "seconds"
				if len(args) >= 3 {
					unit = strings.ToLower(args[2].String())
				}
				switch unit {
				case "ms", "millisecond", "milliseconds":
					return NumberValue{Val: float64(diffDur.Milliseconds())}, nil
				case "s", "sec", "second", "seconds":
					return NumberValue{Val: diffDur.Seconds()}, nil
				case "m", "min", "minute", "minutes":
					return NumberValue{Val: diffDur.Minutes()}, nil
				case "h", "hour", "hours":
					return NumberValue{Val: diffDur.Hours()}, nil
				case "d", "day", "days":
					return NumberValue{Val: diffDur.Hours() / 24.0}, nil
				default:
					return NumberValue{Val: diffDur.Seconds()}, nil
				}
			},
			"Diff": func(args []Value) (Value, error) {
				return dateUnit.Methods["diff"](args)
			},
			"year": func(args []Value) (Value, error) {
				t, err := parseTimeToTime(nil)
				if len(args) > 0 {
					t, err = parseTimeToTime(args[0])
					if err != nil {
						return NilValue{}, err
					}
				}
				return NumberValue{Val: float64(t.Year())}, nil
			},
			"Year": func(args []Value) (Value, error) {
				return dateUnit.Methods["year"](args)
			},
			"month": func(args []Value) (Value, error) {
				t, err := parseTimeToTime(nil)
				if len(args) > 0 {
					t, err = parseTimeToTime(args[0])
					if err != nil {
						return NilValue{}, err
					}
				}
				return NumberValue{Val: float64(t.Month())}, nil
			},
			"Month": func(args []Value) (Value, error) {
				return dateUnit.Methods["month"](args)
			},
			"day": func(args []Value) (Value, error) {
				t, err := parseTimeToTime(nil)
				if len(args) > 0 {
					t, err = parseTimeToTime(args[0])
					if err != nil {
						return NilValue{}, err
					}
				}
				return NumberValue{Val: float64(t.Day())}, nil
			},
			"Day": func(args []Value) (Value, error) {
				return dateUnit.Methods["day"](args)
			},
		},
		Directives: make(map[string]NativeDirective),
	}
	return dateUnit
}

func parseTimeToTime(v Value) (time.Time, error) {
	if v == nil || v == (NilValue{}) {
		return time.Now(), nil
	}
	switch val := v.(type) {
	case NumberValue:
		n := int64(val.Val)
		if n > 100000000000 {
			return time.UnixMilli(n), nil
		}
		return time.Unix(n, 0), nil
	case StringValue:
		str := strings.TrimSpace(val.Val)
		if str == "" || strings.EqualFold(str, "now") {
			return time.Now(), nil
		}
		formats := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
			"2006-01-02",
			"02/01/2006",
			"02/01/2006 15:04:05",
			"15:04:05",
		}
		for _, f := range formats {
			if t, err := time.Parse(f, str); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("date: unable to parse date string '%s'", str)
	case MapValue:
		if isoVal, ok := val.Pairs["iso"]; ok {
			return time.Parse(time.RFC3339, isoVal.String())
		}
		if tsVal, ok := val.Pairs["timestamp"]; ok {
			if nVal, ok := tsVal.(NumberValue); ok {
				return time.UnixMilli(int64(nVal.Val)), nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("date: invalid date value of type %s", v.Type())
}

func dateToMap(t time.Time) MapValue {
	return MapValue{
		Pairs: map[string]Value{
			"iso":       StringValue{Val: t.Format(time.RFC3339)},
			"year":      NumberValue{Val: float64(t.Year())},
			"month":     NumberValue{Val: float64(t.Month())},
			"day":       NumberValue{Val: float64(t.Day())},
			"hour":      NumberValue{Val: float64(t.Hour())},
			"minute":    NumberValue{Val: float64(t.Minute())},
			"second":    NumberValue{Val: float64(t.Second())},
			"timestamp": NumberValue{Val: float64(t.UnixMilli())},
			"weekday":   StringValue{Val: t.Weekday().String()},
		},
	}
}

func convertDateFormat(layout string) string {
	r := layout
	r = strings.ReplaceAll(r, "YYYY", "2006")
	r = strings.ReplaceAll(r, "yyyy", "2006")
	r = strings.ReplaceAll(r, "YY", "06")
	r = strings.ReplaceAll(r, "MM", "01")
	r = strings.ReplaceAll(r, "DD", "02")
	r = strings.ReplaceAll(r, "dd", "02")
	r = strings.ReplaceAll(r, "HH", "15")
	r = strings.ReplaceAll(r, "hh", "03")
	r = strings.ReplaceAll(r, "mm", "04")
	r = strings.ReplaceAll(r, "ss", "05")
	r = strings.ReplaceAll(r, "SSS", "000")
	return r
}
