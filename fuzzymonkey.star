def Env(env): return 'implme' #deleteme
if Env('SUT_IS_PROD') != '':
    # pass
    spec = 'openapi3'
else:
    spec = 'betting'
print('Now testing {}.'.format(spec))

bearerAuth = Env("MYCOMPANY_AUTH_TOKEN") or fail("unset MYCOMPANY_AUTH_TOKEN")
bearerAuth = 'Bearer ' + bearerAuth
backendHost = Env("MYCOMPANY_HOST_URL") or fail("unset MYCOMPANY_HOST_URL")

# OpenAPIv3(
#     file = '../../../_panda/pandapi.git/schemas/'+spec+'.json',

#     host = backendHost,
#     authorization = bearerAuth,

#     exec_reset = 'printf "Resetting state...\n"'
# )


Spec(
    model = OpenAPIv3(file='../../../_panda/pandapi.git/schemas/openapi3.json'),
    overrides = {
        'host': backendHost,
        'authorization': bearerAuth,
    },
)

# print()
# def main():
#     MyState = {'thing': 1}
#     def f(x):
#         a = MyState['thing']
#         MyState['thing'] += x
#         b = MyState['thing']*MyState['thing']
#         print(a, '-->', b)
#         return b
#     return f
# mn = main()
# print(mn(1), mn(1)) # "1 4 9 16"

# MyState = {'thing': 1}
# def f(x):
#     a = MyState['thing']
#     MyState['thing'] += x
#     b = MyState['thing']*MyState['thing']
#     print(a, '-->', b)
#     return b
# f(1); f(1)

SUT(
    start = ['true'],
    reset = ['true'],
    stop  = ['true'],
)

# Oracle(
#     start = ['true'],
#     reset = ['true'],
#     stop  = ['true'],
# )


#weapons = {}
State = {
    'weapons': {},
}

def actionAfterWeapons(response):
    print(StateGet())
    print("!!! actionAfterWeapons", response)
    return
    #assert.eq(response['headers']['Content-Type'], 'application/json')
    body = response['body']['as_json']
    # No need: response has been validated and JSON decoded
    #assert.true(body)
    # Set some state
    weapons = StateGet('weapons')
    weapons[ body['id'] ] = body
    StateUpdate('weapons', weapons)

def actionAfterGetExistingWeapon(response):
    print('actionAfterGetWeapon', response)
    weapon_id = int(response['request']['url'][-1])
    body = response['body']['as_json']
    # Ensure an API contract
    #assert.eq(weapon_id, body['id'])
    # Implied: if weapon_id in StateGet('weapons'):
    # Verify state
    #AssertThat(body).equals(weapons[weapon_id])
    if body != StateGet('weapons')[weapon_id]:
        fail("wrong data for weapon:", weapon_id,
             "expected", StateGet('weapons')[weapon_id],
             "got", body)

# def unpure():
#     BigBoyState['blip'] = 42.0

# someState = {'a': 'b'}
# def maybeUnpure(d):
#     d['a'] = 'c'
#     return d
# print('>>>>>', someState, maybeUnpure(someState), someState)

# There MUST NOT be any upper case exports.
# State = {"strk": v0} # : State is optional but HAS TO be a Dict.
# StateSet(k, v)
# StateUpdate(k, v2)
# StateGet(k)
# StateItems()
# StateKeys()
# StateDelete(k)

#TriggerActionAfterProbe(
After(
    probe = 'monkey:http:response',
    predicate = None,
    match = {
        'request': {'method': 'GET', 'path': '/csgo/weapons'},
        'status_code': 200,
    },
    action = actionAfterWeapons,
)

After(
    probe = 'monkey:http:response',
    predicate = lambda response: all([
        response['request']['method'] == 'GET',
        response['request']['route'] == '/csgo/weapons/:weapon_id',
        response['status_code'] in range(200, 299),
        response['body']['id'] in State['weapons'],
    ]),
    match = None,
    action = actionAfterGetExistingWeapon,
)
