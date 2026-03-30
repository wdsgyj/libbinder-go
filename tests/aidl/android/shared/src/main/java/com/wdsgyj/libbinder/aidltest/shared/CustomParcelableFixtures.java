package com.wdsgyj.libbinder.aidltest.shared;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;

public final class CustomParcelableFixtures {
    private CustomParcelableFixtures() {
    }

    public static CustomBox input() {
        CustomBox out = new CustomBox();
        out.id = 7;
        out.label = "seed";
        out.tags = new ArrayList<>();
        out.tags.add("red");
        out.tags.add("blue");
        out.meta = new HashMap<>();
        out.meta.put("left", "west");
        out.meta.put("right", "east");
        return out;
    }

    public static CustomBox normalize(String prefix, CustomBox value) {
        if (value == null) {
            return null;
        }
        CustomBox out = new CustomBox();
        out.id = value.id + 1;
        out.label = value.label == null ? null : prefix + ":" + value.label;
        out.tags = new ArrayList<>();
        if (value.tags != null) {
            for (String tag : value.tags) {
                out.tags.add(prefix + ":" + tag);
            }
        }
        out.meta = new HashMap<>();
        if (value.meta != null) {
            for (Map.Entry<String, String> entry : value.meta.entrySet()) {
                out.meta.put(entry.getKey(), prefix + ":" + entry.getValue());
            }
        }
        return out;
    }

    public static void verifyService(ICustomParcelableService service, String prefix) throws Exception {
        CustomBox input = input();
        CustomBox want = normalize(prefix, input);
        CustomBox got = service.Normalize(input);
        assertTrue("Normalize", CustomBox.equalsValue(want, got));
        if (service.NormalizeNullable(null) != null) {
            throw new IllegalStateException("NormalizeNullable(null) mismatch");
        }
        CustomBox gotNullable = service.NormalizeNullable(input);
        assertTrue("NormalizeNullable", CustomBox.equalsValue(want, gotNullable));
    }

    private static void assertTrue(String name, boolean ok) {
        if (!ok) {
            throw new IllegalStateException(name + " mismatch");
        }
    }
}
