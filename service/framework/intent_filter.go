package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type PatternMatcher struct {
	Pattern       string
	Type          int32
	ParsedPattern []int32
}

type IntentFilterAuthorityEntry struct {
	OriginalHost string
	Host         string
	Wild         bool
	Port         int32
}

type UriRelativeFilter struct {
	URIPart     int32
	PatternType int32
	Filter      string
}

type UriRelativeFilterGroup struct {
	Action  int32
	Filters []UriRelativeFilter
}

type IntentFilter struct {
	Actions                 []string
	Categories              []string
	DataSchemes             []string
	StaticDataTypes         []string
	DataTypes               []string
	MIMEGroups              []string
	DataSchemeSpecificParts []PatternMatcher
	DataAuthorities         []IntentFilterAuthorityEntry
	DataPaths               []PatternMatcher
	Priority                int32
	HasStaticPartialTypes   bool
	HasDynamicPartialTypes  bool
	AutoVerify              bool
	InstantAppVisibility    int32
	Order                   int32
	Extras                  *PersistableBundle
	URIRelativeFilterGroups []UriRelativeFilterGroup
}

func WriteIntentFilterToParcel(p *api.Parcel, v IntentFilter) error {
	actions := v.Actions
	if actions == nil {
		actions = []string{}
	}
	if err := writeJavaStringSliceToParcel(p, actions); err != nil {
		return err
	}
	if err := writeOptionalJavaStringSliceToParcel(p, v.Categories); err != nil {
		return err
	}
	if err := writeOptionalJavaStringSliceToParcel(p, v.DataSchemes); err != nil {
		return err
	}
	if err := writeOptionalJavaStringSliceToParcel(p, v.StaticDataTypes); err != nil {
		return err
	}
	if err := writeOptionalJavaStringSliceToParcel(p, v.DataTypes); err != nil {
		return err
	}
	if err := writeOptionalJavaStringSliceToParcel(p, v.MIMEGroups); err != nil {
		return err
	}
	if err := writePatternMatcherListToParcel(p, v.DataSchemeSpecificParts); err != nil {
		return err
	}
	if err := writeIntentFilterAuthorityEntryListToParcel(p, v.DataAuthorities); err != nil {
		return err
	}
	if err := writePatternMatcherListToParcel(p, v.DataPaths); err != nil {
		return err
	}
	if err := p.WriteInt32(v.Priority); err != nil {
		return err
	}
	if err := p.WriteInt32(boolToInt32(v.HasStaticPartialTypes)); err != nil {
		return err
	}
	if err := p.WriteInt32(boolToInt32(v.HasDynamicPartialTypes)); err != nil {
		return err
	}
	if err := p.WriteInt32(boolToInt32(v.AutoVerify)); err != nil {
		return err
	}
	if err := p.WriteInt32(v.InstantAppVisibility); err != nil {
		return err
	}
	if err := p.WriteInt32(v.Order); err != nil {
		return err
	}
	if v.Extras != nil {
		if err := p.WriteInt32(1); err != nil {
			return err
		}
		if err := WritePersistableBundleToParcel(p, *v.Extras); err != nil {
			return err
		}
	} else {
		if err := p.WriteInt32(0); err != nil {
			return err
		}
	}
	return writeURIRelativeFilterGroupListToParcel(p, v.URIRelativeFilterGroups)
}

func ReadIntentFilterFromParcel(p *api.Parcel) (IntentFilter, error) {
	actions, err := readJavaStringSliceFromParcel(p)
	if err != nil {
		return IntentFilter{}, err
	}
	if actions == nil {
		actions = []string{}
	}
	categories, err := readOptionalJavaStringSliceFromParcel(p)
	if err != nil {
		return IntentFilter{}, err
	}
	dataSchemes, err := readOptionalJavaStringSliceFromParcel(p)
	if err != nil {
		return IntentFilter{}, err
	}
	staticDataTypes, err := readOptionalJavaStringSliceFromParcel(p)
	if err != nil {
		return IntentFilter{}, err
	}
	dataTypes, err := readOptionalJavaStringSliceFromParcel(p)
	if err != nil {
		return IntentFilter{}, err
	}
	mimeGroups, err := readOptionalJavaStringSliceFromParcel(p)
	if err != nil {
		return IntentFilter{}, err
	}
	dataSchemeSpecificParts, err := readPatternMatcherListFromParcel(p)
	if err != nil {
		return IntentFilter{}, err
	}
	dataAuthorities, err := readIntentFilterAuthorityEntryListFromParcel(p)
	if err != nil {
		return IntentFilter{}, err
	}
	dataPaths, err := readPatternMatcherListFromParcel(p)
	if err != nil {
		return IntentFilter{}, err
	}
	priority, err := p.ReadInt32()
	if err != nil {
		return IntentFilter{}, err
	}
	hasStaticPartialTypes, err := p.ReadInt32()
	if err != nil {
		return IntentFilter{}, err
	}
	hasDynamicPartialTypes, err := p.ReadInt32()
	if err != nil {
		return IntentFilter{}, err
	}
	autoVerify, err := p.ReadInt32()
	if err != nil {
		return IntentFilter{}, err
	}
	instantAppVisibility, err := p.ReadInt32()
	if err != nil {
		return IntentFilter{}, err
	}
	order, err := p.ReadInt32()
	if err != nil {
		return IntentFilter{}, err
	}
	extrasPresent, err := p.ReadInt32()
	if err != nil {
		return IntentFilter{}, err
	}
	var extras *PersistableBundle
	if extrasPresent != 0 {
		value, err := ReadPersistableBundleFromParcel(p)
		if err != nil {
			return IntentFilter{}, err
		}
		extras = &value
	}
	groups, err := readURIRelativeFilterGroupListFromParcel(p)
	if err != nil {
		return IntentFilter{}, err
	}
	return IntentFilter{
		Actions:                 actions,
		Categories:              categories,
		DataSchemes:             dataSchemes,
		StaticDataTypes:         staticDataTypes,
		DataTypes:               dataTypes,
		MIMEGroups:              mimeGroups,
		DataSchemeSpecificParts: dataSchemeSpecificParts,
		DataAuthorities:         dataAuthorities,
		DataPaths:               dataPaths,
		Priority:                priority,
		HasStaticPartialTypes:   hasStaticPartialTypes != 0,
		HasDynamicPartialTypes:  hasDynamicPartialTypes != 0,
		AutoVerify:              autoVerify != 0,
		InstantAppVisibility:    instantAppVisibility,
		Order:                   order,
		Extras:                  extras,
		URIRelativeFilterGroups: groups,
	}, nil
}

func WritePatternMatcherToParcel(p *api.Parcel, v PatternMatcher) error {
	if err := p.WriteString(v.Pattern); err != nil {
		return err
	}
	if err := p.WriteInt32(v.Type); err != nil {
		return err
	}
	return writeJavaInt32SliceToParcel(p, v.ParsedPattern)
}

func ReadPatternMatcherFromParcel(p *api.Parcel) (PatternMatcher, error) {
	pattern, err := p.ReadString()
	if err != nil {
		return PatternMatcher{}, err
	}
	patternType, err := p.ReadInt32()
	if err != nil {
		return PatternMatcher{}, err
	}
	parsedPattern, err := readJavaInt32SliceFromParcel(p)
	if err != nil {
		return PatternMatcher{}, err
	}
	return PatternMatcher{
		Pattern:       pattern,
		Type:          patternType,
		ParsedPattern: parsedPattern,
	}, nil
}

func WriteIntentFilterAuthorityEntryToParcel(p *api.Parcel, v IntentFilterAuthorityEntry) error {
	originalHost := v.OriginalHost
	host := v.Host
	wild := v.Wild
	if originalHost == "" && host != "" {
		if wild {
			originalHost = "*" + host
		} else {
			originalHost = host
		}
	}
	if host == "" && originalHost != "" {
		host = originalHost
		if originalHost[0] == '*' {
			wild = true
			host = originalHost[1:]
		}
	}
	if err := p.WriteString(originalHost); err != nil {
		return err
	}
	if err := p.WriteString(host); err != nil {
		return err
	}
	if err := p.WriteInt32(boolToInt32(wild)); err != nil {
		return err
	}
	return p.WriteInt32(v.Port)
}

func ReadIntentFilterAuthorityEntryFromParcel(p *api.Parcel) (IntentFilterAuthorityEntry, error) {
	originalHost, err := p.ReadString()
	if err != nil {
		return IntentFilterAuthorityEntry{}, err
	}
	host, err := p.ReadString()
	if err != nil {
		return IntentFilterAuthorityEntry{}, err
	}
	wild, err := p.ReadInt32()
	if err != nil {
		return IntentFilterAuthorityEntry{}, err
	}
	port, err := p.ReadInt32()
	if err != nil {
		return IntentFilterAuthorityEntry{}, err
	}
	return IntentFilterAuthorityEntry{
		OriginalHost: originalHost,
		Host:         host,
		Wild:         wild != 0,
		Port:         port,
	}, nil
}

func WriteUriRelativeFilterToParcel(p *api.Parcel, v UriRelativeFilter) error {
	if err := p.WriteInt32(v.URIPart); err != nil {
		return err
	}
	if err := p.WriteInt32(v.PatternType); err != nil {
		return err
	}
	return p.WriteString(v.Filter)
}

func ReadUriRelativeFilterFromParcel(p *api.Parcel) (UriRelativeFilter, error) {
	uriPart, err := p.ReadInt32()
	if err != nil {
		return UriRelativeFilter{}, err
	}
	patternType, err := p.ReadInt32()
	if err != nil {
		return UriRelativeFilter{}, err
	}
	filter, err := p.ReadString()
	if err != nil {
		return UriRelativeFilter{}, err
	}
	return UriRelativeFilter{
		URIPart:     uriPart,
		PatternType: patternType,
		Filter:      filter,
	}, nil
}

func WriteUriRelativeFilterGroupToParcel(p *api.Parcel, v UriRelativeFilterGroup) error {
	if err := p.WriteInt32(v.Action); err != nil {
		return err
	}
	if len(v.Filters) == 0 {
		return p.WriteInt32(0)
	}
	if err := p.WriteInt32(int32(len(v.Filters))); err != nil {
		return err
	}
	for _, filter := range v.Filters {
		if err := WriteUriRelativeFilterToParcel(p, filter); err != nil {
			return err
		}
	}
	return nil
}

func ReadUriRelativeFilterGroupFromParcel(p *api.Parcel) (UriRelativeFilterGroup, error) {
	action, err := p.ReadInt32()
	if err != nil {
		return UriRelativeFilterGroup{}, err
	}
	size, err := p.ReadInt32()
	if err != nil {
		return UriRelativeFilterGroup{}, err
	}
	var filters []UriRelativeFilter
	for i := 0; i < int(size); i++ {
		filter, err := ReadUriRelativeFilterFromParcel(p)
		if err != nil {
			return UriRelativeFilterGroup{}, err
		}
		filters = append(filters, filter)
	}
	return UriRelativeFilterGroup{Action: action, Filters: filters}, nil
}

func writeJavaStringSliceToParcel(p *api.Parcel, values []string) error {
	return api.WriteSlice(p, values, func(p *api.Parcel, value string) error {
		return p.WriteString(value)
	})
}

func readJavaStringSliceFromParcel(p *api.Parcel) ([]string, error) {
	return api.ReadSlice(p, func(p *api.Parcel) (string, error) {
		return p.ReadString()
	})
}

func writeOptionalJavaStringSliceToParcel(p *api.Parcel, values []string) error {
	if values == nil {
		return p.WriteInt32(0)
	}
	if err := p.WriteInt32(1); err != nil {
		return err
	}
	return writeJavaStringSliceToParcel(p, values)
}

func readOptionalJavaStringSliceFromParcel(p *api.Parcel) ([]string, error) {
	present, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}
	if present == 0 {
		return nil, nil
	}
	values, err := readJavaStringSliceFromParcel(p)
	if err != nil {
		return nil, err
	}
	if values == nil {
		return []string{}, nil
	}
	return values, nil
}

func writeJavaInt32SliceToParcel(p *api.Parcel, values []int32) error {
	return api.WriteSlice(p, values, func(p *api.Parcel, value int32) error {
		return p.WriteInt32(value)
	})
}

func readJavaInt32SliceFromParcel(p *api.Parcel) ([]int32, error) {
	return api.ReadSlice(p, func(p *api.Parcel) (int32, error) {
		return p.ReadInt32()
	})
}

func writePatternMatcherListToParcel(p *api.Parcel, values []PatternMatcher) error {
	if len(values) == 0 {
		return p.WriteInt32(0)
	}
	if err := p.WriteInt32(int32(len(values))); err != nil {
		return err
	}
	for _, value := range values {
		if err := WritePatternMatcherToParcel(p, value); err != nil {
			return err
		}
	}
	return nil
}

func readPatternMatcherListFromParcel(p *api.Parcel) ([]PatternMatcher, error) {
	size, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}
	var values []PatternMatcher
	for i := 0; i < int(size); i++ {
		value, err := ReadPatternMatcherFromParcel(p)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func writeIntentFilterAuthorityEntryListToParcel(p *api.Parcel, values []IntentFilterAuthorityEntry) error {
	if len(values) == 0 {
		return p.WriteInt32(0)
	}
	if err := p.WriteInt32(int32(len(values))); err != nil {
		return err
	}
	for _, value := range values {
		if err := WriteIntentFilterAuthorityEntryToParcel(p, value); err != nil {
			return err
		}
	}
	return nil
}

func readIntentFilterAuthorityEntryListFromParcel(p *api.Parcel) ([]IntentFilterAuthorityEntry, error) {
	size, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}
	var values []IntentFilterAuthorityEntry
	for i := 0; i < int(size); i++ {
		value, err := ReadIntentFilterAuthorityEntryFromParcel(p)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func writeURIRelativeFilterGroupListToParcel(p *api.Parcel, values []UriRelativeFilterGroup) error {
	if len(values) == 0 {
		return p.WriteInt32(0)
	}
	if err := p.WriteInt32(int32(len(values))); err != nil {
		return err
	}
	for _, value := range values {
		if err := WriteUriRelativeFilterGroupToParcel(p, value); err != nil {
			return err
		}
	}
	return nil
}

func readURIRelativeFilterGroupListFromParcel(p *api.Parcel) ([]UriRelativeFilterGroup, error) {
	size, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}
	var values []UriRelativeFilterGroup
	for i := 0; i < int(size); i++ {
		value, err := ReadUriRelativeFilterGroupFromParcel(p)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
