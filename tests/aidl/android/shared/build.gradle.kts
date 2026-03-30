plugins {
    id("com.android.library")
}

android {
    namespace = "com.wdsgyj.libbinder.aidltest.shared"
    compileSdk = 33
    buildToolsVersion = "35.0.1"

    defaultConfig {
        minSdk = 30
    }

    sourceSets["main"].aidl.srcDirs("src/main/aidl")
}
