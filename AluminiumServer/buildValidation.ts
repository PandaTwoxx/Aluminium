const VALID_BUILD_SYSTEMS = ['cmake', 'make', 'meson', 'custom', 'none'] as const;
const PACKAGE_NAME_REGEX = /^[a-zA-Z0-9._-]{1,100}$/;
const PACKAGE_VERSION_REGEX = /^[a-zA-Z0-9.+_-]{1,100}$/;
const SAFE_BUILD_FLAGS_REGEX = /^[A-Za-z0-9 _./=+-]{0,200}$/;
const SAFE_SOURCE_DIR_REGEX = /^[A-Za-z0-9._\-/]{0,200}$/;
const SHELL_META_REGEX = /[;&|`$<>\\]/;

export function validatePackageName(value: unknown): value is string {
  return typeof value === 'string' && PACKAGE_NAME_REGEX.test(value);
}

export function validatePackageVersion(value: unknown): value is string {
  return typeof value === 'string' && PACKAGE_VERSION_REGEX.test(value);
}

export function validateBuildFlags(value: unknown): value is string {
  return value === undefined || (typeof value === 'string' && SAFE_BUILD_FLAGS_REGEX.test(value));
}

export function validateSourceDir(value: unknown): value is string {
  if (value === undefined) {
    return true;
  }
  if (typeof value !== 'string' || value.length === 0 || value.length > 200) {
    return false;
  }
  if (value.startsWith('/') || value.includes('..') || value.includes('\\')) {
    return false;
  }
  return SAFE_SOURCE_DIR_REGEX.test(value);
}

function containsAbsolutePath(value: string): boolean {
  return /(^|\s)\//.test(value);
}

export function validateCustomScript(value: unknown): value is string {
  return typeof value === 'string'
    && value.length > 0
    && value.length <= 2000
    && !SHELL_META_REGEX.test(value)
    && !/\r|\n/.test(value)
    && !value.includes('..')
    && !containsAbsolutePath(value);
}

export function isValidBuildSystem(value: unknown): value is 'cmake' | 'make' | 'meson' | 'custom' | 'none' {
  return typeof value === 'string' && VALID_BUILD_SYSTEMS.includes(value as 'cmake' | 'make' | 'meson' | 'custom' | 'none');
}
