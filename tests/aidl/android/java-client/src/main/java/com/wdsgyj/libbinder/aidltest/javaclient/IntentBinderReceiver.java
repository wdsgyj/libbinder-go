package com.wdsgyj.libbinder.aidltest.javaclient;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.os.Bundle;
import android.os.IBinder;
import android.os.RemoteException;
import android.util.Log;
import com.wdsgyj.libbinder.aidltest.shared.IAdvancedCallback;

public final class IntentBinderReceiver extends BroadcastReceiver {
    private static final String EXTRA_TOKEN = "token";
    private static final String EXTRA_CALLBACK = "callback";
    private static final String TAG = "IntentBinderReceiver";

    @Override
    public void onReceive(Context context, Intent intent) {
        if (intent == null) {
            return;
        }

        Bundle extras = intent.getExtras();
        if (extras == null) {
            Log.e(TAG, "INTENT_BINDER_RECEIVER_MISSING_EXTRAS");
            return;
        }

        String token = extras.getString(EXTRA_TOKEN);
        IBinder binder = extras.getBinder(EXTRA_CALLBACK);
        if (token == null || token.isEmpty()) {
            Log.e(TAG, "INTENT_BINDER_RECEIVER_MISSING_TOKEN");
            return;
        }
        if (binder == null) {
            Log.e(TAG, "INTENT_BINDER_RECEIVER_MISSING_BINDER token=" + token);
            return;
        }

        String value = "intent-transport-" + token;
        String descriptor = "<unknown>";
        try {
            descriptor = binder.getInterfaceDescriptor();
            IAdvancedCallback callback = IAdvancedCallback.Stub.asInterface(binder);
            String reply = callback.OnSync(value);
            Log.i(
                    TAG,
                    "INTENT_BINDER_RECEIVER_OK token="
                            + token
                            + " descriptor="
                            + descriptor
                            + " reply="
                            + reply);
        } catch (RemoteException e) {
            Log.e(TAG, "INTENT_BINDER_RECEIVER_REMOTE_ERROR token=" + token, e);
        }
    }
}
