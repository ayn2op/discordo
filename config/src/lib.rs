use figment::{
    Figment,
    providers::{Env, Format, Serialized, Toml},
};
use serde::{Deserialize, Serialize};

const ENV_PREFIX: &str = "DISCORDO_";

#[derive(Default, Serialize, Deserialize)]
pub struct Config {}

impl Config {
    pub fn load() -> figment::Result<Self> {
        Figment::from(Serialized::defaults(Self::default()))
            .merge(Env::prefixed(ENV_PREFIX))
            .merge(Toml::string(""))
            .extract()
    }
}

// pub fn add(left: u64, right: u64) -> u64 {
//     left + right
// }

// #[cfg(test)]
// mod tests {
//     use super::*;

//     #[test]
//     fn it_works() {
//         let result = add(2, 2);
//         assert_eq!(result, 4);
//     }
// }
