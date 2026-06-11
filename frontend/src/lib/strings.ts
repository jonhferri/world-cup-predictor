import type { LanguageCode } from '$lib/language.svelte';

export const strings: Record<
	LanguageCode,
	{
		nav: {
			home: string;
			matchTips: string;
			worldCupTips: string;
			bracket: string;
			leagues: string;
		};
		chrome: {
			settings: string;
			about: string;
			logout: string;
			lightTheme: string;
			darkTheme: string;
			worldCupTheme: string;
			standardTheme: string;
			language: string;
			languageAria: string;
		};
		auth: {
			tagline: string;
			subtitle: string;
			emailLabel: string;
			passwordLabel: string;
			emailPlaceholder: string;
			login: string;
			forgotPassword: string;
			or: string;
			newHere: string;
			createAccount: string;
			google: string;
			wrongCredentials: string;
			googleFailed: string;
		};
		register: {
			title: string;
			subtitle: string;
			nameLabel: string;
			passwordHint: string;
			create: string;
			loginPrompt: string;
			loginLink: string;
			error: string;
			passwordTooShort: string;
		};
		forgotPassword: {
			title: string;
			subtitle: string;
			emailLabel: string;
			send: string;
			success: string;
			back: string;
			error: string;
		};
		resetPassword: {
			title: string;
			subtitle: string;
			newPasswordLabel: string;
			confirmLabel: string;
			update: string;
			success: string;
			back: string;
			errorInvalid: string;
			errorMismatch: string;
			passwordTooShort: string;
			updateError: string;
		};
		search: {
			trigger: string;
			panelAria: string;
			placeholder: string;
			close: string;
			loading: string;
			empty: string;
			noResults: string;
			matches: string;
			teams: string;
			groups: string;
			leagues: string;
			noLeagues: string;
		};
		pwa: {
			installTitle: string;
			installBody: string;
			installButton: string;
			close: string;
			iosTitle: string;
			iosStep1: string;
			iosStep2: string;
			iosStep3: string;
			understood: string;
		};
		introCard: {
			kicker: string;
			title: string;
			body: string;
			leaguesTitle: string;
			leaguesBody: string;
			matchTipsTitle: string;
			matchTipsBody: string;
			worldCupTipsTitle: string;
			worldCupTipsBody: string;
			primaryCta: string;
			secondaryCta: string;
			footnote: string;
			close: string;
			settingsTitle: string;
			settingsBody: string;
			settingsActive: string;
			settingsDismissed: string;
			settingsButton: string;
			settingsSuccess: string;
			settingsLink: string;
		};
		tipCard: {
			lockedResult: string;
			noTipLocked: string;
			showFriendTips: string;
			hideFriendTips: string;
			noFriendTips: string;
			saved: string;
			loading: string;
			stageGroup: string;
			stageOther: string;
			day: string;
			live: string;
			locked: string;
			missing: string;
			result: string;
			goThrough: string;
			penalties: string;
			save: string;
			visiting: string;
			crowdTitle: string;
			crowdEmpty: string;
			crowdHome: string;
			crowdDraw: string;
			crowdAway: string;
			crowdTotal: string;
		};
		playerCard: {
			title: string;
			hitRate: string;
			hitRateSub: string;
			longestStreak: string;
			longestStreakSub: string;
			currentStreak: string;
			largestMiss: string;
			largestMissSub: string;
			noStats: string;
		};
		common: {
			languageName: string;
		};
		odds: {
			sourceOddsApi: string;
			sourceRankings: string;
			toggleToDecimal: string;
			toggleToPct: string;
		};
	}
> = {
	en: {
		nav: {
			home: 'Home',
			matchTips: 'Match Tips',
			worldCupTips: 'Competition Tips',
			bracket: 'Bracket',
			leagues: 'Leagues'
		},
		chrome: {
			settings: 'Settings',
			about: 'About the game',
			logout: 'Log out',
			lightTheme: 'Light theme',
			darkTheme: 'Dark theme',
			worldCupTheme: 'World Cup theme',
			standardTheme: 'Standard theme',
			language: 'English',
			languageAria: 'Language'
		},
		auth: {
			tagline: 'Match tips and World Cup tips in one league.',
			subtitle:
				'Build your crew, pick the games, and follow the World Cup drama from kickoff.',
			emailLabel: 'Email',
			passwordLabel: 'Password',
			emailPlaceholder: 'name@example.com',
			login: 'Log in',
			forgotPassword: 'Forgot password?',
			or: 'OR',
			newHere: 'New here?',
			createAccount: 'Create account.',
			google: 'Continue with Google',
			wrongCredentials: 'Wrong email or password.',
			googleFailed: 'Google sign-in failed.'
		},
		register: {
			title: 'Create account',
			subtitle: 'Join the World Cup tipping competition.',
			nameLabel: 'Display name',
			passwordHint: 'Password must be at least 8 characters.',
			create: 'Create account',
			loginPrompt: 'Already have an account?',
			loginLink: 'Log in',
			error: 'Could not create account.',
			passwordTooShort: 'Password must be at least 8 characters.'
		},
		forgotPassword: {
			title: 'Reset password',
			subtitle: 'Enter the email you signed up with and we will send a reset link.',
			emailLabel: 'Email',
			send: 'Send reset link',
			success: 'If the email is registered, a link is on the way.',
			back: 'Back to sign in',
			error: 'Could not send reset link.'
		},
		resetPassword: {
			title: 'Choose a new password',
			subtitle: 'Enter and confirm your new password.',
			newPasswordLabel: 'New password',
			confirmLabel: 'Confirm new password',
			update: 'Update password',
			success: 'Password updated — sending you to sign in…',
			back: 'Back to sign in',
			errorInvalid: 'The link is invalid or expired.',
			errorMismatch: 'The passwords do not match.',
			passwordTooShort: 'Password must be at least 8 characters.',
			updateError: 'Could not update password.'
		},
		search: {
			trigger: 'Search',
			panelAria: 'Search in Cozinhámos Predictions',
			placeholder: 'Search match, team, group or league',
			close: 'Close search',
			loading: 'Loading search…',
			empty: 'Find a match, team or league.',
			noResults: 'No results.',
			matches: 'Matches',
			teams: 'Teams',
			groups: 'Groups',
			leagues: 'My leagues',
			noLeagues: 'No leagues'
		},
		pwa: {
			installTitle: 'Install Cozinhámos Predictions',
			installBody: 'Home screen icon, full screen, and faster start.',
			installButton: 'Install',
			close: 'Close',
			iosTitle: 'Add Cozinhámos Predictions to the home screen',
			iosStep1: 'Tap the Share button in the Safari toolbar.',
			iosStep2: 'Scroll down and choose Add to Home Screen.',
			iosStep3: 'Tap Add in the top right corner.',
			understood: 'Got it'
		},
		introCard: {
			kicker: 'New here?',
			title: 'Welcome to Cozinhámos Predictions',
			body: 'Pick matches, join leagues, and follow your points through the World Cup.',
			leaguesTitle: 'Leagues',
			leaguesBody: 'Create one or join with a code.',
			matchTipsTitle: 'Match tips',
			matchTipsBody: 'Submit before kickoff.',
			worldCupTipsTitle: 'World Cup tips',
			worldCupTipsBody: 'Pick the bracket and winner.',
			primaryCta: 'Open leagues',
			secondaryCta: 'See match tips',
			footnote: '',
			close: 'Close intro card',
			settingsTitle: 'Welcome card',
			settingsBody:
				'Show the welcome card on the home page again if you want a quick refresher.',
			settingsActive: 'The card is active and will stay on the home page until you close it.',
			settingsDismissed: 'The card is hidden for this user on this device.',
			settingsButton: 'Show the welcome card again',
			settingsSuccess: 'The welcome card is ready to appear on the home page again.',
			settingsLink: 'Go to home'
		},
		tipCard: {
			lockedResult: 'Result',
			noTipLocked: 'No match tip — this game is locked.',
			showFriendTips: 'Show friends’ tips',
			hideFriendTips: 'Hide friends’ tips',
			noFriendTips: 'No friends have picked this match.',
			saved: 'Saved',
			loading: 'Saving…',
			stageGroup: 'Group',
			stageOther: 'Round',
			day: 'Today',
			live: 'Live',
			locked: 'locked',
			missing: 'missing',
			result: 'Your tip',
			goThrough: 'through',
			penalties: 'Penalties',
			save: 'Save',
			visiting: 'Your tip',
			crowdTitle: 'How everyone tipped',
			crowdEmpty: 'No one else has picked this match.',
			crowdHome: 'Home',
			crowdDraw: 'Draw',
			crowdAway: 'Away',
			crowdTotal: 'tips total'
		},
		playerCard: {
			title: 'Player card',
			hitRate: 'Hit rate',
			hitRateSub: 'exact scores',
			longestStreak: 'Longest streak',
			longestStreakSub: 'matches in a row with points',
			currentStreak: 'Current streak',
			largestMiss: 'Biggest miss',
			largestMissSub: 'You tipped',
			noStats: 'No scored matches yet.'
		},
		common: {
			languageName: 'English'
		},
		odds: {
			sourceOddsApi: 'Betting odds',
			sourceRankings: 'FIFA ranking',
			toggleToDecimal: 'Show odds',
			toggleToPct: 'Show %'
		}
	}
};
