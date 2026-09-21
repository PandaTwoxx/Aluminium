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
assert(!validateBuildFlags('; rm -rf /'));

assert(validateSourceDir('https://example.com/project.tar.gz'));
assert(validateSourceDir('https://example.com/build.tar.gz'));
assert(validateSourceDir('https://example.com/project/archive.tar.gz'));
assert(validateSourceDir('git@github.com:owner/repo.git'));
assert(!validateSourceDir('/etc/passwd'));
assert(!validateSourceDir('../evil'));
assert(!validateSourceDir('good\\windows'));

assert(validateCustomScript('ninja -C build'));
assert(!validateCustomScript('sudo rm -rf /'));
assert(!validateCustomScript('eval "echo bad"'));
assert(!validateCustomScript('source /etc/profile'));

assert(isValidBuildSystem('cmake'));
assert(isValidBuildSystem('custom'));
assert(!isValidBuildSystem('unsupported'));

console.log('All build validation tests passed.');
