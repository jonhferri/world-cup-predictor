export type LanguageCode = 'nb' | 'nn' | 'en';

export function isLanguageCode(value: unknown): value is LanguageCode {
	return value === 'nb' || value === 'nn' || value === 'en';
}

class LanguageStore {
	code = $state<LanguageCode>('en');

	get resolved(): LanguageCode {
		return 'en';
	}

	get locale() {
		return 'en-US';
	}

	get isEnglish() {
		return true;
	}

	get isBokmal() {
		return false;
	}

	get isNynorsk() {
		return false;
	}

	// Always returns the English variant regardless of arguments.
	text<T>(_nb: T, _nn: T, en: T): T {
		return en;
	}

	set(_next: LanguageCode) {}
	toggle() {}
}

export const language = new LanguageStore();
