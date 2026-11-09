// Package objcx is Glutter's thin layer over purego's objc package (SPEC §29.2): selectors made once,
// Foundation values (NSString, NSNumber, NSArray, NSURL, NSError) to and from Go, autorelease pools,
// blocks, and the registry of the Objective-C classes that Go implements (the app delegate, the view,
// the accessibility elements), whose instances find their Go object by a handle.
//
// The rules of the package are those of SPEC §29.2:
//   - Objects that a function makes and a caller must release are made by functions named New...
//     (they return a +1 reference); the others return objects that the caller does not own.
//   - Every call that makes Objective-C objects runs inside WithPool, which locks the goroutine to its
//     thread for the time it takes, because an autorelease pool belongs to a thread.
//   - Classes are made once per process, with the Glutter prefix and the process id in their name.
//     Their methods are purego callbacks, counted in package dl (a process can have about 2,000).
//
// The package has files for darwin only; this file keeps it listed on every OS.
//
// Flutter: no counterpart.
package objcx
