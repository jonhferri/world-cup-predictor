export type RuntimeLanguageCode = 'en';

export function readRuntimeLanguage(): RuntimeLanguageCode {
	return 'en';
}

export function isRuntimeEnglish() {
	return true;
}

export function runtimeText<T>(_nb: T, _nn: T, en: T): T {
	return en;
}

export function readRuntimeLocale() {
	return 'en-US';
}
