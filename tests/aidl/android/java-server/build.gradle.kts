plugins {
    id("com.android.application")
}

android {
    namespace = "com.wdsgyj.libbinder.aidltest.javaserver"
    compileSdk = 33
    buildToolsVersion = "35.0.1"

    defaultConfig {
        applicationId = "com.wdsgyj.libbinder.aidltest.javaserver"
        minSdk = 30
        targetSdk = 33
        versionCode = 1
        versionName = "0.0.1"
        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }
}

dependencies {
    implementation(project(":shared"))
}
