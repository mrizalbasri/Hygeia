import type { Clerk } from '@clerk/clerk-js';

type SignInOptions = NonNullable<Parameters<Clerk['mountSignIn']>[1]>;

export const authAppearance = {
	variables: {
		colorPrimary: '#0052cc',
		colorBackground: '#ffffff',
		colorDanger: '#b42318',
		fontFamily: "'Plus Jakarta Sans', sans-serif",
		fontSize: '14px',
		borderRadius: '12px'
	},
	elements: {
		rootBox: { width: '100%', minWidth: '0', maxWidth: '100%' },
		cardBox: {
			width: '100%',
			minWidth: '0',
			maxWidth: '100%',
			overflow: 'visible',
			boxShadow: 'none',
			border: 'none',
			background: 'transparent'
		},
		card: {
			padding: '0',
			width: '100%',
			minWidth: '0',
			overflow: 'visible',
			borderRadius: '0',
			boxShadow: 'none',
			border: 'none',
			background: 'transparent',
			gap: '24px'
		},
		logoBox: { display: 'none' },
		header: { alignItems: 'flex-start', textAlign: 'left', gap: '10px' },
		headerTitle: 'hygeia-auth-title',
		headerSubtitle: 'hygeia-auth-subtitle',
		form: { gap: '18px', margin: '0' },
		formField: { gap: '8px' },
		formFieldLabel: { color: '#0a1f44', fontSize: '13px', fontWeight: '600' },
		formFieldInput: 'hygeia-auth-input',
		formButtonPrimary: 'hygeia-auth-submit',
		socialButtonsBlockButton: 'hygeia-auth-social',
		socialButtonsBlockButtonText: { fontSize: '14px', fontWeight: '500' },
		dividerText: { color: '#526176', fontSize: '12px' },
		dividerLine: { background: '#d7e5f5' },
		footer: {
			background: 'transparent',
			backgroundImage: 'none',
			border: 'none',
			padding: '20px 0 0'
		},
		footerAction: { padding: '0', background: 'transparent' },
		footerActionText: { color: '#526176', fontSize: '13px' },
		footerActionLink: { color: '#0052cc', fontSize: '13px', fontWeight: '600' }
	}
} satisfies SignInOptions['appearance'];
