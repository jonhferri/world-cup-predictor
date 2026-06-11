const knockoutLabels: Record<string, string> = {
	R32: 'Round of 32',
	R16: 'Round of 16',
	QF: 'Quarter-finals',
	SF: 'Semi-finals',
	'3RD': 'Third-place play-off',
	FINAL: 'Final'
};

export function stageName(stage: string) {
	return knockoutLabels[stage] ?? stage;
}

export function matchStageLabel(match: { stage: string; groupLetter?: string }) {
	if (match.stage === 'group') {
		return `Group stage · Group ${match.groupLetter ?? ''}`;
	}
	return stageName(match.stage);
}

export const stageOrder = ['R32', 'R16', 'QF', 'SF', '3RD', 'FINAL'];
