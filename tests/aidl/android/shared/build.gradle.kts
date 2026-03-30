plugins {
    id("com.android.library")
}

android {
    namespace = "com.wdsgyj.libbinder.aidltest.shared"
    compileSdk = 35
    buildToolsVersion = "35.0.1"

    defaultConfig {
        minSdk = 30
    }

    buildFeatures {
        aidl = true
    }

    sourceSets["main"].aidl.srcDirs("src/main/aidl")
}
