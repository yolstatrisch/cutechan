"use strict";

const path = require("path");
const del = require("del");
const stripAnsi = require("strip-ansi");
const autoprefixer = require("autoprefixer");
const cssnano = require("cssnano");
const gulp = require("gulp");
const babel = require("gulp-babel");
const gutil = require("gulp-util");
const jsonminify = require("gulp-jsonminify");
const less = require("gulp-less");
const postcss = require("gulp-postcss");
const rename = require("gulp-rename");
const sourcemaps = require("gulp-sourcemaps");
const ts = require("gulp-typescript");
const uglify = require("gulp-uglify");
const notify = require("gulp-notify");
const runSequence = require("run-sequence");

const DIST_DIR = "dist";
const STATIC_DIR = path.join(DIST_DIR, "static");
const JS_DIR = path.join(STATIC_DIR, "js");
const VENDOR_DIR = path.join(JS_DIR, "vendor");
const CSS_DIR = path.join(STATIC_DIR, "css");
const LANG_DIR = path.join(STATIC_DIR, "lang");
const FONTS_DIR = path.join(STATIC_DIR, "fonts");

// Keep script alive and rebuild on file changes.
// Triggered with the `-w` flag.
const watch = gutil.env.w;

// Dependency tasks for the default tasks.
const tasks = [];

// Notify about errors.
const notifyError = notify.onError({
  title: "<%= error.name %>",
  message: "<%= options.stripAnsi(error.message) %>",
  templateOptions: {stripAnsi},
});

// Simply log the error on continuos builds, but fail the build and exit
// with an error status, if failing a one-time build. This way we can
// use failure to build the client to not pass Travis CL tests.
function handleError(err) {
  if (watch) {
    notifyError(err);
    this.emit("end");
  } else {
    throw err;
  }
}

// Create a new gulp task and set it to execute on default and
// incrementally.
function createTask(name, path, task, watchPath) {
  tasks.push(name);
  gulp.task(name, () =>
    task(gulp.src(path))
  );

  // Recompile on source update, if running with the `-w` flag.
  if (watch) {
    gulp.watch(watchPath || path, [name]);
  }
}

function buildClient(outFile) {
  return gulp.src("client/**/*.ts")
    .pipe(sourcemaps.init())
    .pipe(ts.createProject("client/tsconfig.json", {
      outFile,
    })(ts.reporter.nullReporter()))
    .on("error", handleError);
}

// Builds the client files of the appropriate ECMAScript version.
// TODO(Kagami): ES6-aware minifier.
function buildES6() {
  const name = "es6";
  tasks.push(name);
  gulp.task(name, () =>
    buildClient("app.js")
      .pipe(sourcemaps.write("maps"))
      .pipe(gulp.dest(JS_DIR)));

  // Recompile on source update, if running with the `-w` flag.
  if (watch) {
    gulp.watch("client/**/*.ts", [name])
  }
}

// Build legacy ES5 client for old browsers.
// TODO(Kagami): Output to ES5 in TS instead.
function buildES5() {
  const name = "es5";
  tasks.push(name);
  gulp.task(name, () =>
    buildClient("app.es5.js")
      .pipe(babel({
        presets: ["latest"],
      }))
      .pipe(uglify())
      .on("error", handleError)
      .pipe(sourcemaps.write("maps"))
      .pipe(gulp.dest(JS_DIR))
  );
}

gulp.task("clean", () => {
  return del(DIST_DIR);
});

// Client JS files.
buildES6();
buildES5();

// Third-party dependencies.
createTask("vendor", [
  "node_modules/almond/almond.js ",
  "node_modules/babel-polyfill/dist/polyfill.min.js",
  "node_modules/whatwg-fetch/fetch.js ",
  "node_modules/dom4/build/dom4.js",
  "node_modules/core-js/client/core.min.js",
  "node_modules/proxy-polyfill/proxy.min.js",
], src =>
  src
    .pipe(gulp.dest(VENDOR_DIR))
);

// Compile Less to CSS.
createTask("css", ["less/*.less", "!less/*.mix.less"], src => {
  return src
    .pipe(sourcemaps.init())
    .pipe(less())
    .on("error", handleError)
    .pipe(postcss([
      autoprefixer(),
      cssnano({discardComments: {removeAll: true}}),
    ]))
    .pipe(sourcemaps.write("maps"))
    .pipe(gulp.dest(CSS_DIR))
}, "less/*.less");

// Language packs.
createTask("lang", "lang/**/*.json", src =>
  src
    .pipe(jsonminify())
    .on("error", handleError)
    .pipe(gulp.dest(LANG_DIR))
);

// Static assets.
createTask("assets", "assets/**/*", src =>
  src
    .pipe(gulp.dest(STATIC_DIR))
);

// Fonts.
createTask("fonts", "node_modules/font-awesome/fonts/fontawesome-webfont.*",
           src =>
  src
    .pipe(gulp.dest(FONTS_DIR))
);

// Build everything.
gulp.task("default", runSequence("clean", tasks));
