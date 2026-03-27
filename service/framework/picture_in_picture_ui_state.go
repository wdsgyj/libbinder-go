package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type PictureInPictureUiState struct {
	IsStashed            bool
	IsTransitioningToPip bool
}

func WritePictureInPictureUiStateToParcel(p *api.Parcel, v PictureInPictureUiState) error {
	if err := p.WriteBool(v.IsStashed); err != nil {
		return err
	}
	return p.WriteBool(v.IsTransitioningToPip)
}

func ReadPictureInPictureUiStateFromParcel(p *api.Parcel) (PictureInPictureUiState, error) {
	isStashed, err := p.ReadBool()
	if err != nil {
		return PictureInPictureUiState{}, err
	}
	isTransitioningToPip, err := p.ReadBool()
	if err != nil {
		return PictureInPictureUiState{}, err
	}
	return PictureInPictureUiState{
		IsStashed:            isStashed,
		IsTransitioningToPip: isTransitioningToPip,
	}, nil
}

func WriteNullablePictureInPictureUiStateToParcel(p *api.Parcel, v *PictureInPictureUiState) error {
	if v == nil {
		return p.WriteInt32(0)
	}
	if err := p.WriteInt32(1); err != nil {
		return err
	}
	return WritePictureInPictureUiStateToParcel(p, *v)
}

func ReadNullablePictureInPictureUiStateFromParcel(p *api.Parcel) (*PictureInPictureUiState, error) {
	present, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}
	if present == 0 {
		return nil, nil
	}
	v, err := ReadPictureInPictureUiStateFromParcel(p)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
