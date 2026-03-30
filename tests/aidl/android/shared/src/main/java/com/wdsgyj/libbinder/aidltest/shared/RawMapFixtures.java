package com.wdsgyj.libbinder.aidltest.shared;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;

public final class RawMapFixtures {
    private RawMapFixtures() {
    }

    public static Map<Object, Object> input() {
        HashMap<Object, Object> out = new HashMap<>();
        out.put("name", "seed");
        out.put("count", 7);
        out.put("active", true);
        ArrayList<Object> tags = new ArrayList<>();
        tags.add("red");
        tags.add(2);
        tags.add(3L);
        out.put("tags", tags);
        HashMap<Object, Object> meta = new HashMap<>();
        meta.put("mode", "alpha");
        meta.put("enabled", true);
        ArrayList<Object> ids = new ArrayList<>();
        ids.add(1);
        ids.add("two");
        meta.put("ids", ids);
        out.put("meta", meta);
        return out;
    }

    public static Map<Object, Object> normalize(String prefix, Map<?, ?> value) {
        @SuppressWarnings("unchecked")
        Map<Object, Object> out = (Map<Object, Object>) normalizeValue(prefix, value);
        return out;
    }

    public static void verifyService(RawMapServiceProtocol.Service service, String prefix) throws Exception {
        Map<Object, Object> input = input();
        Map<Object, Object> want = normalize(prefix, input);
        Map<Object, Object> got = castMap(service.Normalize(input));
        if (!deepEquals(want, got)) {
            throw new IllegalStateException("Normalize mismatch: want=" + want + " got=" + got);
        }
    }

    private static Object normalizeValue(String prefix, Object value) {
        if (value == null) {
            return null;
        }
        if (value instanceof String) {
            return prefix + ":" + value;
        }
        if (value instanceof Integer) {
            return ((Integer) value) + 1;
        }
        if (value instanceof Long) {
            return ((Long) value) + 1L;
        }
        if (value instanceof Boolean) {
            return value;
        }
        if (value instanceof List) {
            ArrayList<Object> out = new ArrayList<>();
            for (Object item : (List<?>) value) {
                out.add(normalizeValue(prefix, item));
            }
            return out;
        }
        if (value instanceof Map) {
            HashMap<Object, Object> out = new HashMap<>();
            for (Map.Entry<?, ?> entry : ((Map<?, ?>) value).entrySet()) {
                out.put(entry.getKey(), normalizeValue(prefix, entry.getValue()));
            }
            return out;
        }
        return value;
    }

    private static boolean deepEquals(Object left, Object right) {
        if (left == right) {
            return true;
        }
        if (left == null || right == null) {
            return false;
        }
        if (left instanceof Map && right instanceof Map) {
            Map<?, ?> leftMap = (Map<?, ?>) left;
            Map<?, ?> rightMap = (Map<?, ?>) right;
            if (leftMap.size() != rightMap.size()) {
                return false;
            }
            for (Map.Entry<?, ?> entry : leftMap.entrySet()) {
                if (!deepEquals(entry.getValue(), rightMap.get(entry.getKey()))) {
                    return false;
                }
            }
            return true;
        }
        if (left instanceof List && right instanceof List) {
            List<?> leftList = (List<?>) left;
            List<?> rightList = (List<?>) right;
            if (leftList.size() != rightList.size()) {
                return false;
            }
            for (int i = 0; i < leftList.size(); i++) {
                if (!deepEquals(leftList.get(i), rightList.get(i))) {
                    return false;
                }
            }
            return true;
        }
        return Objects.equals(left, right);
    }

    @SuppressWarnings("unchecked")
    private static Map<Object, Object> castMap(Map<?, ?> value) {
        return (Map<Object, Object>) value;
    }
}
