export type RuntimeLanguageCode = 'nb' | 'nn' | 'en';

const STORAGE_KEY = 'language';
const DEFAULT_LANGUAGE: RuntimeLanguageCode = 'nb';

function normalizeRuntimeLanguage(value: string | null | undefined): RuntimeLanguageCode {
	if (value === 'en') return 'en';
	if (value === 'nn') return 'nn';
	return DEFAULT_LANGUAGE;
}

export function readRuntimeLanguage(): RuntimeLanguageCode {
	if (typeof window !== 'undefined') {
		try {
			return normalizeRuntimeLanguage(localStorage.getItem(STORAGE_KEY));
		} catch {
			const htmlLang = document.documentElement.lang.toLowerCase();
			return htmlLang.startsWith('en')
				? 'en'
				: htmlLang.startsWith('nn')
					? 'nn'
					: DEFAULT_LANGUAGE;
		}
	}
	return DEFAULT_LANGUAGE;
}

export function isRuntimeEnglish() {
	return readRuntimeLanguage() === 'en';
}

export function runtimeText<T>(nb: T, nn: T, en: T): T {
	const lang = readRuntimeLanguage();
	if (lang === 'en') return en;
	if (lang === 'nn') return nn;
	return nb;
}

export function readRuntimeLocale() {
	return isRuntimeEnglish() ? 'en-US' : 'no-NO';
}
