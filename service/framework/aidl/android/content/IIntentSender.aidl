package android.content;

import android.content.IIntentReceiver;
import android.content.Intent;
import android.os.Bundle;

oneway interface IIntentSender {
    void send(int code, @nullable in Intent intent, @nullable String resolvedType,
            in IBinder whitelistToken, IIntentReceiver finishedReceiver,
            @nullable String requiredPermission, @nullable in Bundle options);
}
