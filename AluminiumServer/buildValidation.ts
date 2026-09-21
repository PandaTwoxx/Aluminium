const VALID_BUILD_SYSTEMS = ['cmake', 'make', 'meson', 'custom', 'none'] as const;
const PACKAGE_NAME_REGEX = /^[a-zA-Z0-9._-]{1,100}$/;
const PACKAGE_VERSION_REGEX = /^[a-zA-Z0-9.+_-]{1,100}$/;

// Allow common build flag chars including $, (), so callers can write
// things like --with-openssl=$HOME/... or -DFOO=$(shell ...)
const SAFE_BUILD_FLAGS_REGEX = /^[A-Za-z0-9 _./=+()$-]{0,500}$/;

const SAFE_SOURCE_URL_REGEX = /^(https?:\/\/|git@|ssh:\/\/|git:\/\/)[A-Za-z0-9._~:/?#[\]@!$&'()*+,;=%-]+$/i;

// Patterns that are genuinely dangerous in a sandboxed build script context.
// We block specific escalation constructs rather than banning all shell syntax,
// since real build scripts legitimately need &&, $VAR, pipes, redirects, etc.
const DANGEROUS_SCRIPT_PATTERNS: RegExp[] = [
  // eval / exec builtins allow arbitrary code execution past any filtering
  /\beval\b/,
  /\bexec\b/,
  // Prevent writing to sensitive system directories
  />\s*\/etc\//,
  />\s*\/usr\//,
  />\s*\/bin\//,
  />\s*\/sbin\//,
  />\s*\/lib\//,
  />\s*\/System\//,
  // Prevent sourcing arbitrary files from outside the Aluminium tree
  /\bsource\s+(?!\$HOME\/\.aluminium)/,
  /\.\s+(?!\$HOME\/\.aluminium)/,
  // curl/wget piped directly into a shell interpreter
  /\b(?:curl|wget)\b.*\|\s*(?:bash|sh|zsh|python|ruby|perl|node)\b/,
  // Prevent sudo / su escalation
  /\bsudo\b/,
  /\bsu\s/,
];

export function validatePackageName(value: unknown): value is string {
  return typeof value === 'string' && PACKAGE_NAME_REGEX.test(value);
}

export function validatePackageVersion(value: unknown): value is string {
  return typeof value === 'string' && PACKAGE_VERSION_REGEX.test(value);
}

export function validateBuildFlags(value: unknown): value is string {
  return value === undefined || (typeof value === 'string' && SAFE_BUILD_FLAGS_REGEX.test(value));
}

export function validateCustomScript(value: unknown, allowUnrestricted: boolean = false): boolean {
  if (typeof value !== 'string') return false;
  if (value.length > 20000) return false;
  if (allowUnrestricted) return true;

  for (const pattern of DANGEROUS_SCRIPT_PATTERNS) {
    if (pattern.test(value)) return false;
  }

  return true;
}

export function validateSourceDir(value: unknown): value is string {
  if (value === undefined) {
    return true;
  }
  if (typeof value !== 'string' || value.length === 0 || value.length > 500) {
    return false;
  }
  const trimmedValue = value.trim();
  if (trimmedValue.length === 0) {
    return false;
  }
  if (trimmedValue.startsWith('/') || trimmedValue.includes('..') || trimmedValue.includes('\\')) {
    return false;
  }
  return SAFE_SOURCE_URL_REGEX.test(trimmedValue);
}

function containsAbsolutePath(value: string): boolean {
  return /(^|\s)\//.test(value);
}

export function isValidBuildSystem(value: unknown): value is 'cmake' | 'make' | 'meson' | 'custom' | 'none' {
  return typeof value === 'string' && VALID_BUILD_SYSTEMS.includes(value as 'cmake' | 'make' | 'meson' | 'custom' | 'none');
}
