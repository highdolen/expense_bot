package ai

import "encoding/json"

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

var tools = []Tool{
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "create_expense",
			Description: "Создать расход текущего пользователя. Сумма указывается в копейках. Определи по сообщению пользователя намерение относительно даты и передай его в поле date. Не вычисляй конкретную дату самостоятельно. Если дата не указана, используй type=unspecified. Для фразы «2 дня назад» используй type=relative и offset_days=-2. Для конкретной календарной даты используй type=calendar_date и передай день, месяц и год, если он указан. Если год не указан, оставь year отсутствующим: приложение определит его самостоятельно.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"amount": {
						"type": "integer",
						"minimum": 1,
						"description": "Сумма расхода в копейках. Например, 350 рублей — 35000"
					},
					"category": {
						"type": "string",
						"enum": ["food", "transport", "housing", "entertainment", "health", "shopping", "other"],
						"description": "Категория расхода"
					},
					"comment": {
						"type": "string",
						"description": "Краткое описание покупки без суммы и даты"
					},
					"date": {
						"type": "object",
						"description": "Структурированное намерение пользователя относительно даты расхода",
						"properties": {
							"type": {
								"type": "string",
								"enum": ["unspecified", "relative", "calendar_date"],
								"description": "unspecified — дата не указана; relative — дата относительно сегодняшнего дня; calendar_date — конкретная календарная дата"
							},
							"offset_days": {
								"type": "integer",
								"description": "Смещение в календарных днях относительно сегодняшней даты пользователя. Например, -2 для «2 дня назад», -1 для «вчера», 0 для «сегодня», 1 для «завтра». Используется только при type=relative."
							},
							"day": {
								"type": "integer",
								"minimum": 1,
								"maximum": 31,
								"description": "День месяца для конкретной календарной даты"
							},
							"month": {
								"type": "integer",
								"minimum": 1,
								"maximum": 12,
								"description": "Месяц для конкретной календарной даты"
							},
							"year": {
								"type": "integer",
								"minimum": 1,
								"maximum": 9999,
								"description": "Год, если пользователь явно его указал"
							}
						},
						"required": ["type"],
						"additionalProperties": false
					}
				},
				"required": ["amount", "category", "comment", "date"],
				"additionalProperties": false
			}`),
		},
	},
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "list_expenses",
			Description: "Получить расходы текущего пользователя и их общую сумму за период. Выбирай период по смыслу запроса. today — сегодня; yesterday — вчера; day_before_yesterday — позавчера; week — текущая календарная неделя с понедельника по воскресенье; previous_week — предыдущая календарная неделя с понедельника по воскресенье; last_7_days — последние 7 календарных дней, включая сегодня; last_n_days — последние N календарных дней, включая сегодня; month — текущий календарный месяц; last_month — предыдущий календарный месяц; all — все расходы; custom — конкретная дата или произвольный диапазон дат. Для last_n_days обязательно передай days — количество дней из запроса пользователя. Например, «за последние 5 дней» означает period=last_n_days и days=5, «за последние 14 дней» — days=14. Не проси пользователя указывать даты вручную, если он назвал количество дней. Для custom передай from и to в формате YYYY-MM-DD. Для одной даты from и to должны совпадать. Если категория не указана, не передавай category.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"period": {
						"type": "string",
						"enum": [
							"all",
							"today",
							"yesterday",
							"day_before_yesterday",
							"week",
							"previous_week",
							"last_7_days",
							"last_n_days",
							"month",
							"last_month",
							"custom"
						],
						"description": "Период расходов. Для произвольного количества последних дней используй last_n_days и передавай days."
					},
					"days": {
						"type": "integer",
						"minimum": 1,
						"maximum": 3650,
						"description": "Количество последних календарных дней, включая сегодня. Обязательно для period=last_n_days. Например, 5 для запроса «за последние 5 дней»."
					},
					"from": {
						"type": "string",
						"format": "date",
						"description": "Начало произвольного периода включительно в формате YYYY-MM-DD. Используется только для period=custom."
					},
					"to": {
						"type": "string",
						"format": "date",
						"description": "Конец произвольного периода включительно в формате YYYY-MM-DD. Используется только для period=custom."
					},
					"category": {
						"type": "string",
						"enum": ["food", "transport", "housing", "entertainment", "health", "shopping", "other"],
						"description": "Необязательный фильтр категории. Передавай только если пользователь явно указал категорию."
					}
				},
				"required": ["period"],
				"additionalProperties": false
			}`),
		},
	},
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "update_expense",
			Description: "Изменить существующий расход текущего пользователя. Передай id расхода и только те поля, которые требуется изменить. Сумма указывается в копейках. Не изменяй дату, если пользователь не просил её изменить. При явном запросе на изменение даты передай структурированное намерение в поле date, используя те же правила, что и для create_expense.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"id": {
						"type": "integer",
						"minimum": 1,
						"description": "Идентификатор изменяемого расхода"
					},
					"amount": {
						"type": "integer",
						"minimum": 1,
						"description": "Новая сумма расхода в копейках"
					},
					"category": {
						"type": "string",
						"enum": ["food", "transport", "housing", "entertainment", "health", "shopping", "other"],
						"description": "Новая категория расхода"
					},
					"date": {
						"type": "object",
						"description": "Передавай только при явной просьбе изменить дату расхода",
						"properties": {
							"type": {
								"type": "string",
								"enum": ["unspecified", "relative", "calendar_date"],
								"description": "unspecified — дата не указана; relative — относительная дата; calendar_date — конкретная календарная дата"
							},
							"offset_days": {
								"type": "integer",
								"description": "Смещение в календарных днях относительно сегодняшней даты пользователя. Например, -2 для «2 дня назад»."
							},
							"day": {
								"type": "integer",
								"minimum": 1,
								"maximum": 31,
								"description": "День месяца для конкретной даты"
							},
							"month": {
								"type": "integer",
								"minimum": 1,
								"maximum": 12,
								"description": "Месяц для конкретной даты"
							},
							"year": {
								"type": "integer",
								"minimum": 1,
								"maximum": 9999,
								"description": "Год, если пользователь явно его указал"
							}
						},
						"required": ["type"],
						"additionalProperties": false
					},
					"comment": {
						"type": "string",
						"description": "Новое описание покупки"
					}
				},
				"required": ["id"],
				"additionalProperties": false
			}`),
		},
	},
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "delete_expense",
			Description: "Удалить расход текущего пользователя по его идентификатору.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"id": {
						"type": "integer",
						"minimum": 1,
						"description": "Идентификатор удаляемого расхода"
					}
				},
				"required": ["id"],
				"additionalProperties": false
			}`),
		},
	},
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "set_timezone",
			Description: "Установить часовой пояс текущего пользователя. Вызывай этот инструмент только по явной просьбе пользователя изменить часовой пояс.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"timezone": {
						"type": "string",
						"description": "IANA-идентификатор часового пояса, например Europe/Moscow или Europe/Vilnius"
					}
				},
				"required": ["timezone"],
				"additionalProperties": false
			}`),
		},
	},
}
