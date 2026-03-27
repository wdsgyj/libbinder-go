package framework

import api "github.com/wdsgyj/libbinder-go/binder"

type ApplicationErrorReportParcelableCrashInfo struct {
	ExceptionHandlerClassName *string
	ExceptionClassName        *string
	ExceptionMessage          *string
	ThrowFileName             *string
	ThrowClassName            *string
	ThrowMethodName           *string
	ThrowLineNumber           int32
	StackTrace                *string
	CrashTag                  *string
}

func WriteApplicationErrorReportParcelableCrashInfoToParcel(p *api.Parcel, v ApplicationErrorReportParcelableCrashInfo) error {
	if err := p.WriteNullableString(v.ExceptionHandlerClassName); err != nil {
		return err
	}
	if err := p.WriteNullableString(v.ExceptionClassName); err != nil {
		return err
	}
	if err := p.WriteNullableString(v.ExceptionMessage); err != nil {
		return err
	}
	if err := p.WriteNullableString(v.ThrowFileName); err != nil {
		return err
	}
	if err := p.WriteNullableString(v.ThrowClassName); err != nil {
		return err
	}
	if err := p.WriteNullableString(v.ThrowMethodName); err != nil {
		return err
	}
	if err := p.WriteInt32(v.ThrowLineNumber); err != nil {
		return err
	}
	if err := p.WriteNullableString(v.StackTrace); err != nil {
		return err
	}
	return p.WriteNullableString(v.CrashTag)
}

func ReadApplicationErrorReportParcelableCrashInfoFromParcel(p *api.Parcel) (ApplicationErrorReportParcelableCrashInfo, error) {
	exceptionHandlerClassName, err := p.ReadNullableString()
	if err != nil {
		return ApplicationErrorReportParcelableCrashInfo{}, err
	}
	exceptionClassName, err := p.ReadNullableString()
	if err != nil {
		return ApplicationErrorReportParcelableCrashInfo{}, err
	}
	exceptionMessage, err := p.ReadNullableString()
	if err != nil {
		return ApplicationErrorReportParcelableCrashInfo{}, err
	}
	throwFileName, err := p.ReadNullableString()
	if err != nil {
		return ApplicationErrorReportParcelableCrashInfo{}, err
	}
	throwClassName, err := p.ReadNullableString()
	if err != nil {
		return ApplicationErrorReportParcelableCrashInfo{}, err
	}
	throwMethodName, err := p.ReadNullableString()
	if err != nil {
		return ApplicationErrorReportParcelableCrashInfo{}, err
	}
	throwLineNumber, err := p.ReadInt32()
	if err != nil {
		return ApplicationErrorReportParcelableCrashInfo{}, err
	}
	stackTrace, err := p.ReadNullableString()
	if err != nil {
		return ApplicationErrorReportParcelableCrashInfo{}, err
	}
	crashTag, err := p.ReadNullableString()
	if err != nil {
		return ApplicationErrorReportParcelableCrashInfo{}, err
	}
	return ApplicationErrorReportParcelableCrashInfo{
		ExceptionHandlerClassName: exceptionHandlerClassName,
		ExceptionClassName:        exceptionClassName,
		ExceptionMessage:          exceptionMessage,
		ThrowFileName:             throwFileName,
		ThrowClassName:            throwClassName,
		ThrowMethodName:           throwMethodName,
		ThrowLineNumber:           throwLineNumber,
		StackTrace:                stackTrace,
		CrashTag:                  crashTag,
	}, nil
}

type ApplicationExitInfo = OpaqueParcelable

func WriteApplicationExitInfoToParcel(p *api.Parcel, v ApplicationExitInfo) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadApplicationExitInfoFromParcel(p *api.Parcel) (ApplicationExitInfo, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type ApplicationInfo = OpaqueParcelable

func WriteApplicationInfoToParcel(p *api.Parcel, v ApplicationInfo) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadApplicationInfoFromParcel(p *api.Parcel) (ApplicationInfo, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type ApplicationStartInfo = OpaqueParcelable

func WriteApplicationStartInfoToParcel(p *api.Parcel, v ApplicationStartInfo) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadApplicationStartInfoFromParcel(p *api.Parcel) (ApplicationStartInfo, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type AssistContent = OpaqueParcelable

func WriteAssistContentToParcel(p *api.Parcel, v AssistContent) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadAssistContentFromParcel(p *api.Parcel) (AssistContent, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type AssistStructure = OpaqueParcelable

func WriteAssistStructureToParcel(p *api.Parcel, v AssistStructure) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadAssistStructureFromParcel(p *api.Parcel) (AssistStructure, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type BackAnimationAdapter = OpaqueParcelable

func WriteBackAnimationAdapterToParcel(p *api.Parcel, v BackAnimationAdapter) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadBackAnimationAdapterFromParcel(p *api.Parcel) (BackAnimationAdapter, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type BackNavigationInfo = OpaqueParcelable

func WriteBackNavigationInfoToParcel(p *api.Parcel, v BackNavigationInfo) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadBackNavigationInfoFromParcel(p *api.Parcel) (BackNavigationInfo, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type Bitmap = OpaqueParcelable

func WriteBitmapToParcel(p *api.Parcel, v Bitmap) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadBitmapFromParcel(p *api.Parcel) (Bitmap, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type CharSequence struct {
	Text    string
	Spanned bool
}

func WriteCharSequenceToParcel(p *api.Parcel, v CharSequence) error {
	return writeCharSequenceValueToParcel(p, &v)
}

func ReadCharSequenceFromParcel(p *api.Parcel) (CharSequence, error) {
	value, err := readCharSequenceValueFromParcel(p)
	if err != nil {
		return CharSequence{}, err
	}
	if value == nil {
		return CharSequence{}, api.ErrBadParcelable
	}
	return *value, nil
}

func writeCharSequenceValueToParcel(p *api.Parcel, v *CharSequence) error {
	if v == nil {
		if err := p.WriteInt32(1); err != nil {
			return err
		}
		return p.WriteNullableString8(nil)
	}
	if v.Spanned {
		if err := p.WriteInt32(0); err != nil {
			return err
		}
		if err := p.WriteString8(v.Text); err != nil {
			return err
		}
		return p.WriteInt32(0)
	}
	if err := p.WriteInt32(1); err != nil {
		return err
	}
	return p.WriteString8(v.Text)
}

func readCharSequenceValueFromParcel(p *api.Parcel) (*CharSequence, error) {
	kind, err := p.ReadInt32()
	if err != nil {
		return nil, err
	}
	text, err := p.ReadNullableString8()
	if err != nil {
		return nil, err
	}
	if text == nil {
		return nil, nil
	}
	switch kind {
	case 1:
		return &CharSequence{Text: *text}, nil
	case 0:
		spanKind, err := p.ReadInt32()
		if err != nil {
			return nil, err
		}
		if spanKind != 0 {
			return nil, api.ErrBadParcelable
		}
		return &CharSequence{Text: *text, Spanned: true}, nil
	default:
		return nil, api.ErrBadParcelable
	}
}

type Configuration = OpaqueParcelable

func WriteConfigurationToParcel(p *api.Parcel, v Configuration) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadConfigurationFromParcel(p *api.Parcel) (Configuration, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type ContentProviderHolder = OpaqueParcelable

func WriteContentProviderHolderToParcel(p *api.Parcel, v ContentProviderHolder) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadContentProviderHolderFromParcel(p *api.Parcel) (ContentProviderHolder, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type DebugMemoryInfo = OpaqueParcelable

func WriteDebugMemoryInfoToParcel(p *api.Parcel, v DebugMemoryInfo) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadDebugMemoryInfoFromParcel(p *api.Parcel) (DebugMemoryInfo, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type GrantedUriPermission = OpaqueParcelable

func WriteGrantedUriPermissionToParcel(p *api.Parcel, v GrantedUriPermission) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadGrantedUriPermissionFromParcel(p *api.Parcel) (GrantedUriPermission, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type GraphicBuffer = OpaqueParcelable

func WriteGraphicBufferToParcel(p *api.Parcel, v GraphicBuffer) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadGraphicBufferFromParcel(p *api.Parcel) (GraphicBuffer, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type Notification = OpaqueParcelable

func WriteNotificationToParcel(p *api.Parcel, v Notification) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadNotificationFromParcel(p *api.Parcel) (Notification, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type ParceledListSlice = OpaqueParcelable

func WriteParceledListSliceToParcel(p *api.Parcel, v ParceledListSlice) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadParceledListSliceFromParcel(p *api.Parcel) (ParceledListSlice, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type PictureInPictureParams = OpaqueParcelable

func WritePictureInPictureParamsToParcel(p *api.Parcel, v PictureInPictureParams) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadPictureInPictureParamsFromParcel(p *api.Parcel) (PictureInPictureParams, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type ProviderInfo = OpaqueParcelable

func WriteProviderInfoToParcel(p *api.Parcel, v ProviderInfo) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadProviderInfoFromParcel(p *api.Parcel) (ProviderInfo, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type RemoteAnimationAdapter = OpaqueParcelable

func WriteRemoteAnimationAdapterToParcel(p *api.Parcel, v RemoteAnimationAdapter) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadRemoteAnimationAdapterFromParcel(p *api.Parcel) (RemoteAnimationAdapter, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type RemoteAnimationDefinition = OpaqueParcelable

func WriteRemoteAnimationDefinitionToParcel(p *api.Parcel, v RemoteAnimationDefinition) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadRemoteAnimationDefinitionFromParcel(p *api.Parcel) (RemoteAnimationDefinition, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type RemoteCallback = OpaqueParcelable

func WriteRemoteCallbackToParcel(p *api.Parcel, v RemoteCallback) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadRemoteCallbackFromParcel(p *api.Parcel) (RemoteCallback, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type ResolveInfo = OpaqueParcelable

func WriteResolveInfoToParcel(p *api.Parcel, v ResolveInfo) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadResolveInfoFromParcel(p *api.Parcel) (ResolveInfo, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type SplashScreenViewParcelable = OpaqueParcelable

func WriteSplashScreenViewParcelableToParcel(p *api.Parcel, v SplashScreenViewParcelable) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadSplashScreenViewParcelableFromParcel(p *api.Parcel) (SplashScreenViewParcelable, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type StrictModeViolationInfo = OpaqueParcelable

func WriteStrictModeViolationInfoToParcel(p *api.Parcel, v StrictModeViolationInfo) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadStrictModeViolationInfoFromParcel(p *api.Parcel) (StrictModeViolationInfo, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type TaskSnapshot = OpaqueParcelable

func WriteTaskSnapshotToParcel(p *api.Parcel, v TaskSnapshot) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadTaskSnapshotFromParcel(p *api.Parcel) (TaskSnapshot, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type UserInfo = OpaqueParcelable

func WriteUserInfoToParcel(p *api.Parcel, v UserInfo) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadUserInfoFromParcel(p *api.Parcel) (UserInfo, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}

type WorkSource = OpaqueParcelable

func WriteWorkSourceToParcel(p *api.Parcel, v WorkSource) error {
	return writeOpaqueFrameworkParcelableToParcel(p, v)
}

func ReadWorkSourceFromParcel(p *api.Parcel) (WorkSource, error) {
	return readOpaqueFrameworkParcelableFromParcel(p)
}
