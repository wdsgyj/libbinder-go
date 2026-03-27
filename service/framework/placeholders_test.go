package framework

import (
	"reflect"
	"testing"

	api "github.com/wdsgyj/libbinder-go/binder"
)

func TestFrameworkPlaceholderParcelablesRoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		write func(*api.Parcel, OpaqueParcelable) error
		read  func(*api.Parcel) (OpaqueParcelable, error)
	}{
		{
			name:  "application_exit_info",
			write: WriteApplicationExitInfoToParcel,
			read:  ReadApplicationExitInfoFromParcel,
		},
		{
			name:  "application_info",
			write: WriteApplicationInfoToParcel,
			read:  ReadApplicationInfoFromParcel,
		},
		{
			name:  "application_start_info",
			write: WriteApplicationStartInfoToParcel,
			read:  ReadApplicationStartInfoFromParcel,
		},
		{
			name:  "assist_content",
			write: WriteAssistContentToParcel,
			read:  ReadAssistContentFromParcel,
		},
		{
			name:  "assist_structure",
			write: WriteAssistStructureToParcel,
			read:  ReadAssistStructureFromParcel,
		},
		{
			name:  "back_animation_adapter",
			write: WriteBackAnimationAdapterToParcel,
			read:  ReadBackAnimationAdapterFromParcel,
		},
		{
			name:  "back_navigation_info",
			write: WriteBackNavigationInfoToParcel,
			read:  ReadBackNavigationInfoFromParcel,
		},
		{
			name:  "bitmap",
			write: WriteBitmapToParcel,
			read:  ReadBitmapFromParcel,
		},
		{
			name:  "configuration",
			write: WriteConfigurationToParcel,
			read:  ReadConfigurationFromParcel,
		},
		{
			name:  "content_provider_holder",
			write: WriteContentProviderHolderToParcel,
			read:  ReadContentProviderHolderFromParcel,
		},
		{
			name:  "debug_memory_info",
			write: WriteDebugMemoryInfoToParcel,
			read:  ReadDebugMemoryInfoFromParcel,
		},
		{
			name:  "granted_uri_permission",
			write: WriteGrantedUriPermissionToParcel,
			read:  ReadGrantedUriPermissionFromParcel,
		},
		{
			name:  "graphic_buffer",
			write: WriteGraphicBufferToParcel,
			read:  ReadGraphicBufferFromParcel,
		},
		{
			name:  "notification",
			write: WriteNotificationToParcel,
			read:  ReadNotificationFromParcel,
		},
		{
			name:  "parceled_list_slice",
			write: WriteParceledListSliceToParcel,
			read:  ReadParceledListSliceFromParcel,
		},
		{
			name:  "picture_in_picture_params",
			write: WritePictureInPictureParamsToParcel,
			read:  ReadPictureInPictureParamsFromParcel,
		},
		{
			name:  "provider_info",
			write: WriteProviderInfoToParcel,
			read:  ReadProviderInfoFromParcel,
		},
		{
			name:  "remote_animation_adapter",
			write: WriteRemoteAnimationAdapterToParcel,
			read:  ReadRemoteAnimationAdapterFromParcel,
		},
		{
			name:  "remote_animation_definition",
			write: WriteRemoteAnimationDefinitionToParcel,
			read:  ReadRemoteAnimationDefinitionFromParcel,
		},
		{
			name:  "remote_callback",
			write: WriteRemoteCallbackToParcel,
			read:  ReadRemoteCallbackFromParcel,
		},
		{
			name:  "resolve_info",
			write: WriteResolveInfoToParcel,
			read:  ReadResolveInfoFromParcel,
		},
		{
			name:  "splash_screen_view_parcelable",
			write: WriteSplashScreenViewParcelableToParcel,
			read:  ReadSplashScreenViewParcelableFromParcel,
		},
		{
			name:  "strict_mode_violation_info",
			write: WriteStrictModeViolationInfoToParcel,
			read:  ReadStrictModeViolationInfoFromParcel,
		},
		{
			name:  "task_snapshot",
			write: WriteTaskSnapshotToParcel,
			read:  ReadTaskSnapshotFromParcel,
		},
		{
			name:  "user_info",
			write: WriteUserInfoToParcel,
			read:  ReadUserInfoFromParcel,
		},
		{
			name:  "work_source",
			write: WriteWorkSourceToParcel,
			read:  ReadWorkSourceFromParcel,
		},
	}

	for i, tc := range cases {
		p := api.NewParcel()
		want := NewOpaqueParcelable([]byte{byte(i), 0x7f, 0x00, 0xff})
		if err := tc.write(p, want); err != nil {
			t.Fatalf("%s Write...ToParcel: %v", tc.name, err)
		}
		if err := p.SetPosition(0); err != nil {
			t.Fatalf("%s SetPosition: %v", tc.name, err)
		}
		got, err := tc.read(p)
		if err != nil {
			t.Fatalf("%s Read...FromParcel: %v", tc.name, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s got = %#v, want %#v", tc.name, got, want)
		}
	}
}
