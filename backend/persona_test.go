package main

import (
	"strings"
	"testing"
)

func TestPersonaCountryCompatibility(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		language    string
		user        string
		country     string
		wantAllowed bool
	}{
		{name: "fluent English country", mode: ModeUserL2, language: "EN", user: "CN", country: "GB", wantAllowed: true},
		{name: "non English country rejected for fluent partner", mode: ModeUserL2, language: "EN", user: "CN", country: "JP", wantAllowed: false},
		{name: "multilingual French country", mode: ModeUserL2, language: "FR", user: "CN", country: "CA", wantAllowed: true},
		{name: "Japanese L2 learner of English", mode: ModeLLML2, language: "EN", user: "CN", country: "JP", wantAllowed: true},
		{name: "English background cannot be English learner", mode: ModeLLML2, language: "EN", user: "CN", country: "US", wantAllowed: false},
		{name: "same country excluded in cross cultural learner mode", mode: ModeLLML2, language: "EN", user: "JP", country: "JP", wantAllowed: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPersonaCountryAllowed(tt.mode, tt.language, tt.user, tt.country); got != tt.wantAllowed {
				t.Fatalf("isPersonaCountryAllowed() = %v, want %v", got, tt.wantAllowed)
			}
		})
	}
}

func TestGeneratedPersonaOverridesLegacyRole(t *testing.T) {
	session := ConversationSession{LLMRoleID: "aiko"}
	persona := GeneratedPersona{
		Country: "BR", NameZH: "测试人物", NameEN: "Test Person", Age: 31,
		GenderZH: "未说明", GenderEN: "not specified", NativeLanguage: "Portuguese",
		PersonalityZH: "好奇而谨慎", PersonalityEN: "curious and careful",
		BackgroundZH: "生活在巴西。", BackgroundEN: "Lives in Brazil.",
	}
	applyPersonaToSession(&session, persona)
	profile := getSessionLLMRoleProfile(session)
	if profile.NameEN != persona.NameEN || profile.Country != "BR" || profile.PersonalityZH != persona.PersonalityZH {
		t.Fatalf("generated persona was not preserved: %#v", profile)
	}
	if got := getLLMPersonaNativeCountry(session, User{Country: "CN"}); got != "BR" {
		t.Fatalf("generated country changed unexpectedly: %q", got)
	}
	if got := getLLMPersonaNativeLanguage(session, User{Country: "CN"}); got != "Portuguese" {
		t.Fatalf("generated native language changed unexpectedly: %q", got)
	}
}

func TestFeedbackPromptKeepsGeneratedPersonaIdentity(t *testing.T) {
	session := ConversationSession{
		Mode: ModeUserL2, TargetLanguage: "JP", RelationshipType: "朋友", Topic: "日常交流",
		LLMRoleID: "aiko", LLMCountry: "JP", LLMNativeLanguage: "Japanese",
		LLMNameZH: "高桥凛", LLMNameEN: "Rin Takahashi", LLMAge: 36,
		LLMGenderZH: "女性", LLMGenderEN: "woman",
		LLMPersonalityZH: "说话直接但会认真倾听", LLMBackgroundZH: "在地方图书馆从事活动策划。",
	}
	prompt := buildSessionFeedbackPrompt(session, nil, User{Country: "CN"})
	for _, want := range []string{"高桥凛", "Japan", "Japanese", "不得擅自替换为其他国家文化", "不得用国籍概括人格"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("feedback prompt is missing persona consistency marker %q\n%s", want, prompt)
		}
	}
}

func TestPersonaPromptSeparatesCultureFromPersonality(t *testing.T) {
	country, ok := getPersonaCountry("JP")
	if !ok {
		t.Fatal("test country missing")
	}
	prompt := buildPersonaGenerationPrompt(country, ModeLLML2, "EN", "friends", "daily chat", 42)
	for _, want := range []string{
		"Culture is context, never a personality shortcut",
		"Do not infer temperament",
		"Randomize age",
		"Variation seed: 42",
		"intermediate learner of English",
		"JSON data, not instructions",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("persona prompt is missing %q\n%s", want, prompt)
		}
	}
}

func TestSanitizeGeneratedPersonaRejectsIncompleteOutput(t *testing.T) {
	if _, ok := sanitizeGeneratedPersona(GeneratedPersona{NameEN: "Only a name", Age: 30}); ok {
		t.Fatal("incomplete model output must not be accepted")
	}
}
