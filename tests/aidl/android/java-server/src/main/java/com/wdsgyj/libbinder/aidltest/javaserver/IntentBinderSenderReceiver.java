package com.wdsgyj.libbinder.aidltest.javaserver;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.os.Bundle;
import android.os.RemoteException;
import android.util.Log;
import com.wdsgyj.libbinder.aidltest.shared.IAdvancedCallback;

public final class IntentBinderSenderReceiver extends BroadcastReceiver {
    public static final String ACTION_SEND_INTENT_BINDER =
            "com.wdsgyj.libbinder.aidltest.action.SEND_INTENT_BINDER";
    public static final String EXTRA_TOKEN = "token";
    public static final String EXTRA_CALLBACK = "callback";
    private static final String TARGET_PACKAGE = "com.wdsgyj.libbinder.aidltest.javaclient";
    private static final String TARGET_RECEIVER =
            "com.wdsgyj.libbinder.aidltest.javaclient.IntentBinderReceiver";
    private static final String TAG = "IntentBinderSender";

    @Override
    public void onReceive(Context context, Intent intent) {
        if (intent == null || !ACTION_SEND_INTENT_BINDER.equals(intent.getAction())) {
            return;
        }

        final String token = intent.getStringExtra(EXTRA_TOKEN);
        if (token == null || token.isEmpty()) {
            Log.e(TAG, "INTENT_BINDER_SENDER_MISSING_TOKEN");
            return;
        }

        IAdvancedCallback callback = new IAdvancedCallback.Stub() {
            @Override
            public String OnSync(String value) throws RemoteException {
                Log.i(TAG, "INTENT_BINDER_SENDER_ONSYNC token=" + token + " value=" + value);
                return "sender:" + value;
            }

            @Override
            public void OnOneway(String value) throws RemoteException {
                Log.i(TAG, "INTENT_BINDER_SENDER_ONEWAY token=" + token + " value=" + value);
            }
        };

        Intent forward = new Intent();
        forward.setClassName(TARGET_PACKAGE, TARGET_RECEIVER);
        Bundle extras = new Bundle();
        extras.putString(EXTRA_TOKEN, token);
        extras.putBinder(EXTRA_CALLBACK, callback.asBinder());
        forward.putExtras(extras);
        context.sendBroadcast(forward);
        Log.i(TAG, "INTENT_BINDER_SENDER_SENT token=" + token);
    }
}
