package web

type GeneratorDoc struct {
	Category, Name, ResultType, Description, Example string
	Available                                        bool
}

func generatorDocs() []GeneratorDoc {
	return []GeneratorDoc{
		{"Core", "generate.uuid", "string", "UUID v4. Identical expressions share one ID within a response.", "{\"id\":\"{{ generate.uuid }}\"}", true},
		{"Core", "generate.timestamp", "string", "Current UTC time. Remains live even with a fixed seed.", "{\"created_at\":\"{{ generate.timestamp }}\"}", true},
		{"Core", "generate.integer", "number", "Inclusive integer bounds from -1,000,000,000 to 1,000,000,000.", "{\"stock\":\"{{ generate.integer: [1, 100] }}\"}", true},
		{"Core", "generate.choice", "any JSON value", "Choose from 1–100 JSON values. Escape inner quotes in the body editor.", "{\n  \"status\": \"{{ generate.choice: [\\\"pending\\\", \\\"paid\\\"] }}\"\n}", true},
		{"Person", "generate.person.first_name", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.person.first_name }}\"\n}", false},
		{"Person", "generate.person.last_name", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.person.last_name }}\"\n}", false},
		{"Person", "generate.person.full_name", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.person.full_name }}\"\n}", false},
		{"Internet", "generate.internet.username", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.internet.username }}\"\n}", false},
		{"Internet", "generate.internet.email", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.internet.email }}\"\n}", false},
		{"Internet", "generate.internet.url", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.internet.url }}\"\n}", false},
		{"Internet", "generate.internet.domain", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.internet.domain }}\"\n}", false},
		{"Phone", "generate.phone.number", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.phone.number }}\"\n}", false},
		{"Address", "generate.address.street", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.address.street }}\"\n}", false},
		{"Address", "generate.address.city", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.address.city }}\"\n}", false},
		{"Address", "generate.address.state", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.address.state }}\"\n}", false},
		{"Address", "generate.address.postal_code", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.address.postal_code }}\"\n}", false},
		{"Address", "generate.address.country", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.address.country }}\"\n}", false},
		{"Address", "generate.address.country_code", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.address.country_code }}\"\n}", false},
		{"Company", "generate.company.name", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.company.name }}\"\n}", false},
		{"Company", "generate.company.job_title", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.company.job_title }}\"\n}", false},
		{"Commerce", "generate.commerce.product_name", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.commerce.product_name }}\"\n}", false},
		{"Commerce", "generate.commerce.category", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.commerce.category }}\"\n}", false},
		{"Commerce", "generate.commerce.description", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.commerce.description }}\"\n}", false},
		{"Commerce", "generate.commerce.price", "number", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.commerce.price: {\\\"min\\\":10,\\\"max\\\":500,\\\"decimals\\\":2} }}\"\n}", false},
		{"Commerce", "generate.commerce.currency", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.commerce.currency }}\"\n}", false},
		{"Text", "generate.text.word", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.text.word }}\"\n}", false},
		{"Text", "generate.text.sentence", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.text.sentence }}\"\n}", false},
		{"Text", "generate.text.paragraph", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.text.paragraph }}\"\n}", false},
		{"Date", "generate.date.between", "string", "Proposed generator. Not available in the current runtime.", "{\n  \"value\": \"{{ generate.date.between: {\\\"from\\\":\\\"2025-01-01T00:00:00Z\\\",\\\"to\\\":\\\"2025-12-31T23:59:59Z\\\"} }}\"\n}", false},
	}
}
