package framework

import (
	"fmt"
	"sync/atomic"

	api "github.com/wdsgyj/libbinder-go/binder"
)

var intentRedirectProtectionEnabled atomic.Bool

type Intent struct {
	Action          *string
	Data            *URI
	MIMEType        *string
	Identifier      *string
	Flags           int32
	ExtendedFlags   int32
	Package         *string
	Component       *ComponentName
	SourceBounds    *Rect
	Categories      []string
	Selector        *Intent
	HasClipData     bool
	ContentUserHint int32
	Extras          *Bundle
	OriginalIntent  *Intent
}

func SetIntentRedirectProtectionEnabled(enabled bool) {
	intentRedirectProtectionEnabled.Store(enabled)
}

func WriteIntentToParcel(p *api.Parcel, v Intent) error {
	if err := p.WriteNullableString8(v.Action); err != nil {
		return err
	}
	if err := WriteNullableURIToParcel(p, v.Data); err != nil {
		return err
	}
	if err := p.WriteNullableString8(v.MIMEType); err != nil {
		return err
	}
	if err := p.WriteNullableString8(v.Identifier); err != nil {
		return err
	}
	if err := p.WriteInt32(v.Flags); err != nil {
		return err
	}
	if err := p.WriteInt32(v.ExtendedFlags); err != nil {
		return err
	}
	if err := p.WriteNullableString8(v.Package); err != nil {
		return err
	}
	if err := WriteNullableComponentNameToParcel(p, v.Component); err != nil {
		return err
	}
	if v.SourceBounds != nil {
		if err := p.WriteInt32(1); err != nil {
			return err
		}
		if err := WriteRectToParcel(p, *v.SourceBounds); err != nil {
			return err
		}
	} else {
		if err := p.WriteInt32(0); err != nil {
			return err
		}
	}
	if len(v.Categories) != 0 {
		if err := p.WriteInt32(int32(len(v.Categories))); err != nil {
			return err
		}
		for _, category := range v.Categories {
			if err := p.WriteString8(category); err != nil {
				return err
			}
		}
	} else {
		if err := p.WriteInt32(0); err != nil {
			return err
		}
	}
	if v.Selector != nil {
		if err := p.WriteInt32(1); err != nil {
			return err
		}
		if err := WriteIntentToParcel(p, *v.Selector); err != nil {
			return err
		}
	} else {
		if err := p.WriteInt32(0); err != nil {
			return err
		}
	}
	if v.HasClipData {
		return fmt.Errorf("%w: clip data is not implemented in framework.Intent", api.ErrUnsupported)
	}
	if err := p.WriteInt32(0); err != nil {
		return err
	}
	if err := p.WriteInt32(v.ContentUserHint); err != nil {
		return err
	}
	if err := WriteBundleValueToParcel(p, v.Extras); err != nil {
		return err
	}
	if v.OriginalIntent != nil {
		if err := p.WriteInt32(1); err != nil {
			return err
		}
		if err := WriteIntentToParcel(p, *v.OriginalIntent); err != nil {
			return err
		}
	} else {
		if err := p.WriteInt32(0); err != nil {
			return err
		}
	}
	if intentRedirectProtectionEnabled.Load() {
		if err := p.WriteInt32(0); err != nil {
			return err
		}
	}
	return nil
}

func ReadIntentFromParcel(p *api.Parcel) (Intent, error) {
	action, err := p.ReadNullableString8()
	if err != nil {
		return Intent{}, err
	}
	data, err := ReadNullableURIFromParcel(p)
	if err != nil {
		return Intent{}, err
	}
	mimeType, err := p.ReadNullableString8()
	if err != nil {
		return Intent{}, err
	}
	identifier, err := p.ReadNullableString8()
	if err != nil {
		return Intent{}, err
	}
	flags, err := p.ReadInt32()
	if err != nil {
		return Intent{}, err
	}
	extendedFlags, err := p.ReadInt32()
	if err != nil {
		return Intent{}, err
	}
	pkg, err := p.ReadNullableString8()
	if err != nil {
		return Intent{}, err
	}
	component, err := ReadNullableComponentNameFromParcel(p)
	if err != nil {
		return Intent{}, err
	}
	var sourceBounds *Rect
	hasSourceBounds, err := p.ReadInt32()
	if err != nil {
		return Intent{}, err
	}
	if hasSourceBounds != 0 {
		rect, err := ReadRectFromParcel(p)
		if err != nil {
			return Intent{}, err
		}
		sourceBounds = &rect
	}
	categoryCount, err := p.ReadInt32()
	if err != nil {
		return Intent{}, err
	}
	var categories []string
	if categoryCount > 0 {
		categories = make([]string, 0, int(categoryCount))
		for i := 0; i < int(categoryCount); i++ {
			category, err := p.ReadString8()
			if err != nil {
				return Intent{}, err
			}
			categories = append(categories, category)
		}
	}
	var selector *Intent
	hasSelector, err := p.ReadInt32()
	if err != nil {
		return Intent{}, err
	}
	if hasSelector != 0 {
		value, err := ReadIntentFromParcel(p)
		if err != nil {
			return Intent{}, err
		}
		selector = &value
	}
	hasClipData, err := p.ReadInt32()
	if err != nil {
		return Intent{}, err
	}
	if hasClipData != 0 {
		return Intent{}, fmt.Errorf("%w: clip data is not implemented in framework.Intent", api.ErrUnsupported)
	}
	contentUserHint, err := p.ReadInt32()
	if err != nil {
		return Intent{}, err
	}
	extras, err := ReadBundleValueFromParcel(p)
	if err != nil {
		return Intent{}, err
	}
	var originalIntent *Intent
	hasOriginalIntent, err := p.ReadInt32()
	if err != nil {
		return Intent{}, err
	}
	if hasOriginalIntent != 0 {
		value, err := ReadIntentFromParcel(p)
		if err != nil {
			return Intent{}, err
		}
		originalIntent = &value
	}
	if intentRedirectProtectionEnabled.Load() {
		hasCreatorTokenInfo, err := p.ReadInt32()
		if err != nil {
			return Intent{}, err
		}
		if hasCreatorTokenInfo != 0 {
			return Intent{}, fmt.Errorf("%w: creator token info is not implemented in framework.Intent", api.ErrUnsupported)
		}
	}
	return Intent{
		Action:          action,
		Data:            data,
		MIMEType:        mimeType,
		Identifier:      identifier,
		Flags:           flags,
		ExtendedFlags:   extendedFlags,
		Package:         pkg,
		Component:       component,
		SourceBounds:    sourceBounds,
		Categories:      categories,
		Selector:        selector,
		ContentUserHint: contentUserHint,
		Extras:          extras,
		OriginalIntent:  originalIntent,
	}, nil
}
