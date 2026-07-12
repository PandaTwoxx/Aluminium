import assert from 'node:assert';
import {
  validatePackageName,
  validatePackageVersion,
  validateBuildFlags,
  validateSourceDir,
  validateCustomScript,
  isValidBuildSystem,
} from '../buildValidation.js';

console.log('Running build validation tests...');

assert(validatePackageName('example-package'));
assert(!validatePackageName('bad/package'));
assert(!validatePackageName('')); 
assert(!validatePackageName('name;rm -rf /'));

assert(validatePackageVersion('1.0.0'));
assert(validatePackageVersion('1.0.0-alpha'));
assert(!validatePackageVersion('1.0.0\nrm -rf /'));
assert(!validatePackageVersion('')); 

assert(validateBuildFlags('CFLAGS=-O2'));
assert(validateBuildFlags('')); 
assert(!validateBuildFlags('&& rm -rf /'));
assert(!validateBuildFlags('$(rm -rf /)'));

assert(validateSourceDir('src/project'));
assert(validateSourceDir('build'));
assert(!validateSourceDir('/etc/passwd'));
assert(!validateSourceDir('../evil'));
assert(!validateSourceDir('good\\windows'));

assert(validateCustomScript('ninja -C build'));
assert(!validateCustomScript('rm -rf /'));
assert(!validateCustomScript('echo hi; rm -rf /'));
assert(!validateCustomScript('echo hi\nrm -rf /'));

assert(isValidBuildSystem('cmake'));
assert(isValidBuildSystem('custom'));
assert(!isValidBuildSystem('unsupported'));

console.log('All build validation tests passed.');
